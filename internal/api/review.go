package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *App) review(w http.ResponseWriter, r *http.Request, reviewer int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "提交编号无效")
		return
	}
	var body struct {
		Status  string `json:"status"`
		Comment string `json:"reviewComment"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		bad(w, "请求无效")
		return
	}
	status := strings.ToUpper(body.Status)
	if status != "APPROVED" && status != "REJECTED" {
		bad(w, "审核状态无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var user, task int64
	var old sql.NullString
	var oldReviewer sql.NullInt64
	var submission string
	var count int
	err = tx.QueryRowContext(r.Context(), `SELECT user_id,task_id,status,reviewer_id,COALESCE(submission_data,'') FROM user_task_submissions WHERE id=?`, id).Scan(&user, &task, &old, &oldReviewer, &submission)
	if err == sql.ErrNoRows {
		bad(w, "提交不存在")
		return
	}
	if err != nil {
		fail(w)
		return
	}
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM mentorships WHERE mentor_id=? AND apprentice_id=? AND status='ACTIVE'`, reviewer, user).Scan(&count); err != nil {
		fail(w)
		return
	}
	if count == 0 {
		if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM users reviewer JOIN users student ON student.id=? WHERE reviewer.id=? AND student.role='EXTERNAL' AND reviewer.role IN ('INTERNAL','TEACHER') AND COALESCE(reviewer.level,0)>COALESCE(student.level,0)`, user, reviewer).Scan(&count); err != nil {
			fail(w)
			return
		}
		if count == 0 {
			reply(w, 403, nil, "无权审核该提交")
			return
		}
	}
	if old.String != "PENDING" {
		if oldReviewer.Valid && oldReviewer.Int64 == reviewer && old.String == status {
			ok(w, map[string]any{"id": id, "userId": user, "taskId": task, "status": old.String})
			return
		}
		bad(w, "该提交已审核，不能修改结果；请提交新作业")
		return
	}
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions WHERE user_id=? AND task_id=? AND xp_granted=1`, user, task).Scan(&count); err != nil {
		fail(w)
		return
	}
	grant := status == "APPROVED" && count == 0
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	_, err = tx.ExecContext(r.Context(), `UPDATE user_task_submissions SET status=?,review_comment=?,reviewer_id=?,reviewed_at=?,xp_granted=CASE WHEN ? THEN 1 ELSE xp_granted END WHERE id=?`, status, body.Comment, reviewer, now, grant, id)
	if err != nil {
		fail(w)
		return
	}
	var levelID sql.NullInt64
	var xp int
	if err = tx.QueryRowContext(r.Context(), `SELECT level_id,xp_reward FROM tasks WHERE id=?`, task).Scan(&levelID, &xp); err != nil {
		fail(w)
		return
	}
	if grant {
		if _, err = tx.ExecContext(r.Context(), `UPDATE users SET total_xp=total_xp+? WHERE id=?`, xp, user); err != nil {
			fail(w)
			return
		}
	}
	if status == "APPROVED" && levelID.Valid {
		if err = updateProgress(r.Context(), tx, user, levelID.Int64, now); err != nil {
			fail(w)
			return
		}
		if err = completeLevel(r.Context(), tx, user, levelID.Int64, now); err != nil {
			fail(w)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id, "userId": user, "taskId": task, "submissionData": submission, "status": status, "reviewerId": reviewer, "reviewComment": body.Comment, "reviewedAt": a.timestamp(sql.NullString{String: now, Valid: true})})
}
func updateProgress(ctx context.Context, tx *sql.Tx, user, level int64, now string) error {
	var total, approved int
	if err := tx.QueryRowContext(ctx, `SELECT count(*),COALESCE(sum(CASE WHEN EXISTS(SELECT 1 FROM user_task_submissions s WHERE s.task_id=t.id AND s.user_id=? AND s.status='APPROVED') THEN 1 ELSE 0 END),0) FROM tasks t WHERE t.level_id=?`, user, level).Scan(&total, &approved); err != nil {
		return err
	}
	if total == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned) VALUES(?,?,'LEARNING',?,0) ON CONFLICT(level_id,user_id) DO UPDATE SET progress=CASE WHEN status='COMPLETED' THEN progress ELSE excluded.progress END`, level, user, approved*100/total)
	return err
}
func completeLevel(ctx context.Context, tx *sql.Tx, user, level int64, now string) error {
	var xp, total, approved int
	var mapID sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT xp_reward,map_id FROM levels WHERE id=?`, level).Scan(&xp, &mapID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT count(*),COALESCE(sum(CASE WHEN EXISTS(SELECT 1 FROM user_task_submissions s WHERE s.task_id=t.id AND s.user_id=? AND s.status='APPROVED') THEN 1 ELSE 0 END),0) FROM tasks t WHERE t.level_id=?`, user, level).Scan(&total, &approved); err != nil {
		return err
	}
	if total == 0 || total != approved {
		return nil
	}
	var state sql.NullString
	var earned int
	err = tx.QueryRowContext(ctx, `SELECT status,xp_earned FROM user_level_progress WHERE user_id=? AND level_id=?`, user, level).Scan(&state, &earned)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if state.String == "COMPLETED" {
		return nil
	}
	if earned == 0 && xp > 0 {
		earned = xp
		if _, err = tx.ExecContext(ctx, `UPDATE users SET total_xp=total_xp+? WHERE id=?`, xp, user); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned,started_at,completed_at) VALUES(?,?,'COMPLETED',100,?,?,?) ON CONFLICT(level_id,user_id) DO UPDATE SET status='COMPLETED',progress=100,xp_earned=excluded.xp_earned,started_at=COALESCE(started_at,excluded.started_at),completed_at=excluded.completed_at`, level, user, earned, now, now)
	if err != nil {
		return err
	}
	if mapID.Valid {
		var levels, done int
		if err = tx.QueryRowContext(ctx, `SELECT count(*),COALESCE(sum(CASE WHEN p.status='COMPLETED' THEN 1 ELSE 0 END),0) FROM levels l LEFT JOIN user_level_progress p ON p.level_id=l.id AND p.user_id=? WHERE l.map_id=?`, user, mapID.Int64).Scan(&levels, &done); err != nil {
			return err
		}
		var mapProgressID int64
		err = tx.QueryRowContext(ctx, `SELECT id FROM user_map_progress WHERE user_id=? AND map_id=? LIMIT 1`, user, mapID.Int64).Scan(&mapProgressID)
		complete := levels > 0 && done >= levels
		if err == sql.ErrNoRows {
			_, err = tx.ExecContext(ctx, `INSERT INTO user_map_progress(user_id,map_id,levels_completed,total_levels,is_completed,completed_at) VALUES(?,?,?,?,?,CASE WHEN ? THEN ? ELSE NULL END)`, user, mapID.Int64, done, levels, complete, complete, now)
		} else if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE user_map_progress SET levels_completed=?,total_levels=?,is_completed=?,completed_at=CASE WHEN ? THEN COALESCE(completed_at,?) ELSE completed_at END WHERE id=?`, done, levels, complete, complete, now, mapProgressID)
		}
		if err != nil {
			return err
		}
	}
	var done, oldLevel int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM user_level_progress WHERE user_id=? AND status='COMPLETED'`, user).Scan(&done); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(level,0) FROM users WHERE id=?`, user).Scan(&oldLevel); err != nil {
		return err
	}
	newLevel := (done*70 + 41) / 42
	if newLevel != oldLevel {
		if _, err = tx.ExecContext(ctx, `UPDATE users SET level=? WHERE id=?`, newLevel, user); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE mentorships SET status='ENDED' WHERE status='ACTIVE' AND ((mentor_id=? AND EXISTS(SELECT 1 FROM users WHERE id=apprentice_id AND COALESCE(level,0)>=?)) OR (apprentice_id=? AND EXISTS(SELECT 1 FROM users WHERE id=mentor_id AND COALESCE(level,0)<=?)))`, user, newLevel, user, newLevel)
		if err != nil {
			return err
		}
	}
	return nil
}
func (a *App) complete(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "关卡编号无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	if err = completeLevel(r.Context(), tx, user, id, now); err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, nil)
}
