package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) submit(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "任务编号无效")
		return
	}
	var req struct {
		SubmissionData string `json:"submissionData"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.SubmissionData) == "" {
		bad(w, "提交内容不能为空")
		return
	}
	req.SubmissionData = normalizeContent(req.SubmissionData)
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var levelID sql.NullInt64
	err = tx.QueryRowContext(r.Context(), `SELECT level_id FROM tasks WHERE id=?`, id).Scan(&levelID)
	if err == sql.ErrNoRows {
		bad(w, "任务不存在")
		return
	}
	if err != nil {
		fail(w)
		return
	}
	var count int
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions WHERE user_id=? AND task_id=? AND status='APPROVED'`, user, id).Scan(&count); err != nil {
		fail(w)
		return
	}
	if count > 0 {
		bad(w, "该任务已通过")
		return
	}
	var role string
	if err = tx.QueryRowContext(r.Context(), `SELECT COALESCE(role,'') FROM users WHERE id=?`, user).Scan(&role); err != nil {
		fail(w)
		return
	}
	if role != "EXTERNAL" {
		if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM mentorships WHERE apprentice_id=? AND status='ACTIVE'`, user).Scan(&count); err != nil {
			fail(w)
			return
		}
		if count == 0 {
			bad(w, "你还没有绑定师傅，无法提交人工审核。请先在师徒系统中绑定师傅。")
			return
		}
	}
	if levelID.Valid {
		l, exists := s.byID[levelID.Int64]
		if exists && role != "TEACHER" {
			for _, pid := range l.Prereqs {
				if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM user_level_progress WHERE user_id=? AND level_id=? AND status='COMPLETED'`, user, pid).Scan(&count); err != nil {
					fail(w)
					return
				}
				if count == 0 {
					bad(w, "前置关卡未通关，无法提交该关卡的任务")
					return
				}
			}
		}
		now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
		_, err = tx.ExecContext(r.Context(), `INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned,started_at) VALUES(?,?,'LEARNING',1,0,?) ON CONFLICT(level_id,user_id) DO UPDATE SET status=CASE WHEN status='LOCKED' OR started_at IS NULL THEN 'LEARNING' ELSE status END,progress=CASE WHEN progress=0 THEN 1 ELSE progress END,started_at=COALESCE(started_at,excluded.started_at)`, levelID.Int64, user, now)
		if err != nil {
			fail(w)
			return
		}
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	res, err := tx.ExecContext(r.Context(), `INSERT INTO user_task_submissions(user_id,task_id,submission_data,status,submitted_at,ai_review_count) VALUES(?,?,?,'PENDING',?,0)`, user, id, req.SubmissionData, now)
	if err != nil {
		fail(w)
		return
	}
	newID, err := res.LastInsertId()
	if err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": newID, "userId": user, "taskId": id, "submissionData": req.SubmissionData, "status": "PENDING", "submittedAt": a.timestamp(sql.NullString{String: now, Valid: true}), "aiReviewCount": 0, "aiFeedback": nil, "reviewComment": nil, "reviewerId": nil, "reviewedAt": nil, "xpGranted": nil})
}

func (a *App) mySubmissions(w http.ResponseWriter, r *http.Request, user int64) {
	page, size := 0, 10
	var err error
	if s := r.URL.Query().Get("page"); s != "" {
		page, err = strconv.Atoi(s)
		if err != nil || page < 0 {
			bad(w, "页码无效")
			return
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		size, err = strconv.Atoi(s)
		if err != nil || size < 1 || size > 100 {
			bad(w, "每页数量无效")
			return
		}
	}
	if page > 1000000 {
		bad(w, "页码无效")
		return
	}
	var total int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions WHERE user_id=?`, user).Scan(&total); err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT s.id,s.task_id,s.user_id,s.submission_data,s.status,s.ai_feedback,s.review_comment,s.submitted_at,s.reviewed_at,COALESCE(t.title,''),u.username,u.name FROM user_task_submissions s LEFT JOIN tasks t ON t.id=s.task_id JOIN users u ON u.id=s.user_id WHERE s.user_id=? ORDER BY s.submitted_at DESC,s.id DESC LIMIT ? OFFSET ?`, user, size, page*size)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var sid, tid, uid int64
		var data, status, feedback, comment, submitted, reviewed, name sql.NullString
		var title, username string
		if err = rows.Scan(&sid, &tid, &uid, &data, &status, &feedback, &comment, &submitted, &reviewed, &title, &username, &name); err != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": sid, "taskId": tid, "taskTitle": title, "userId": uid, "username": username, "name": nullString(name), "submissionData": signedContent(data), "status": nullString(status), "aiFeedback": nullString(feedback), "reviewComment": nullString(comment), "submittedAt": a.timestamp(submitted), "reviewedAt": a.timestamp(reviewed)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"content": out, "total": total})
}
func (a *App) submissions(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "任务编号无效")
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT s.id,s.task_id,s.user_id,s.submission_data,s.status,s.ai_feedback,s.review_comment,s.submitted_at,s.reviewed_at,COALESCE(t.title,''),u.username,u.name FROM user_task_submissions s LEFT JOIN tasks t ON t.id=s.task_id JOIN users u ON u.id=s.user_id WHERE s.user_id=? AND s.task_id=? ORDER BY s.submitted_at DESC,s.id DESC`, user, id)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var sid, tid, uid int64
		var data, status, feedback, comment, submitted, reviewed, name sql.NullString
		var title, username string
		if err = rows.Scan(&sid, &tid, &uid, &data, &status, &feedback, &comment, &submitted, &reviewed, &title, &username, &name); err != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": sid, "taskId": tid, "taskTitle": title, "userId": uid, "username": username, "name": nullString(name), "submissionData": signedContent(data), "status": nullString(status), "aiFeedback": nullString(feedback), "reviewComment": nullString(comment), "submittedAt": a.timestamp(submitted), "reviewedAt": a.timestamp(reviewed)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, out)
}
