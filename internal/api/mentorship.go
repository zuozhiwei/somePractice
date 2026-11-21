package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const eligibleMentorPair = `apprentice.role='INTERNAL'
	AND mentor.role IN ('INTERNAL','TEACHER','SUPERADMIN')
	AND COALESCE(mentor.level,0)>COALESCE(apprentice.level,0)`

const existingMentorPair = `EXISTS (
	SELECT 1 FROM mentorships WHERE status IN ('ACTIVE','PENDING')
		AND ((mentor_id=mentor.id AND apprentice_id=apprentice.id)
			OR (mentor_id=apprentice.id AND apprentice_id=mentor.id))
)`

const eligiblePendingMentorship = `EXISTS (
	SELECT 1 FROM users AS mentor JOIN users AS apprentice ON apprentice.id=m.apprentice_id
	WHERE mentor.id=m.mentor_id AND ` + eligibleMentorPair + `
)`

type mentorInviteClaims struct {
	// Login tokens include userId and must never be accepted as invitations.
	LoginUserID *int64 `json:"userId,omitempty"`
	Version     int64  `json:"sv,omitempty"`
	jwt.RegisteredClaims
}

func (a *App) inviteLink(w http.ResponseWriter, r *http.Request, user int64) {
	var eligible bool
	var version int64
	if a.DB.QueryRowContext(r.Context(), `SELECT
		CASE WHEN u.role IN ('INTERNAL','TEACHER','SUPERADMIN') AND COALESCE(u.level,0)>0 THEN 1 ELSE 0 END,
		COALESCE(v.version,0) FROM users u LEFT JOIN login_versions v ON v.user_id=u.id WHERE u.id=?`, user).Scan(&eligible, &version) != nil {
		fail(w)
		return
	}
	if !eligible {
		bad(w, "内部成员和教师达到 1 级后可邀请徒弟")
		return
	}
	now := time.Now()
	token, err := jwt.NewWithClaims(a.signingMethod(), mentorInviteClaims{Version: version, RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(user, 10), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour))}}).SignedString(a.key)
	if err != nil {
		fail(w)
		return
	}
	ok(w, token)
}
func (a *App) mentorshipRequest(w http.ResponseWriter, r *http.Request, user int64) {
	var req struct {
		MentorID int64 `json:"mentorId"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.MentorID <= 0 || req.MentorID == user {
		bad(w, "师父无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var eligible, existing bool
	err = tx.QueryRowContext(r.Context(), `SELECT COALESCE((`+eligibleMentorPair+`),0), `+existingMentorPair+`
		FROM users AS mentor JOIN users AS apprentice ON apprentice.id=? WHERE mentor.id=?`, user, req.MentorID).Scan(&eligible, &existing)
	if err == sql.ErrNoRows {
		bad(w, "用户不存在")
		return
	}
	if err != nil {
		fail(w)
		return
	}
	if !eligible {
		bad(w, "仅内部成员可拜师，师父等级须高于申请人")
		return
	}
	if existing {
		bad(w, "师徒关系已存在或申请处理中")
		return
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	res, err := tx.Exec(`INSERT INTO mentorships(mentor_id,apprentice_id,status,created_at) VALUES(?,?,'PENDING',?)`, req.MentorID, user, now)
	if err != nil {
		fail(w)
		return
	}
	id, _ := res.LastInsertId()
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id, "mentorId": req.MentorID, "apprenticeId": user, "status": "PENDING", "createdAt": a.timestamp(sql.NullString{String: now, Valid: true})})
}
func (a *App) mentorshipAction(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "关系编号无效")
		return
	}
	action := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	var q string
	rejected := "关系不存在或无权操作"
	switch action {
	case "approve":
		q = `UPDATE mentorships AS m SET status='ACTIVE'
			WHERE id=? AND mentor_id=? AND status='PENDING' AND ` + eligiblePendingMentorship
		rejected = "无法通过此申请。师父等级须高于申请人"
	case "reject":
		q = `UPDATE mentorships SET status='ENDED' WHERE id=? AND mentor_id=? AND status='PENDING'`
	case "quit":
		q = `UPDATE mentorships SET status='ENDED' WHERE id=? AND apprentice_id=? AND status='ACTIVE'`
	case "cancel":
		q = `DELETE FROM mentorships WHERE id=? AND apprentice_id=? AND status='PENDING'`
	default:
		bad(w, "操作无效")
		return
	}
	res, err := a.DB.ExecContext(r.Context(), q, id, user)
	if err != nil {
		fail(w)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		bad(w, rejected)
		return
	}
	ok(w, nil)
}
func (a *App) relationList(r *http.Request, user int64, kind string) ([]map[string]any, error) {
	var where, peer string
	approval := "0"
	switch kind {
	case "mentors":
		where = "m.apprentice_id=? AND m.status='ACTIVE'"
		peer = "m.mentor_id"
	case "apprentices":
		where = "m.mentor_id=? AND m.status='ACTIVE'"
		peer = "m.apprentice_id"
	case "pending":
		where = "m.mentor_id=? AND m.status='PENDING'"
		peer = "m.apprentice_id"
		approval = eligiblePendingMentorship
	case "sentRequests":
		where = "m.apprentice_id=? AND m.status='PENDING'"
		peer = "m.mentor_id"
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT m.id,u.id,u.username,u.name,u.avatar,m.status,m.created_at,COALESCE(u.level,0),`+approval+
		` FROM mentorships m JOIN users u ON u.id=`+peer+` WHERE `+where+` ORDER BY m.id`, user)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		var id, uid int64
		var username, status string
		var name, avatar, created sql.NullString
		var level int
		var canApprove bool
		if err = rows.Scan(&id, &uid, &username, &name, &avatar, &status, &created, &level, &canApprove); err != nil {
			rows.Close()
			return nil, err
		}
		entry := map[string]any{"id": id, "userId": uid, "username": username, "name": nullString(name), "avatar": nullString(avatar), "status": status, "createdAt": a.timestamp(created), "level": level, "completedLevels": 0, "totalLevels": 0, "currentLevelIndex": 0, "currentLevelId": nil, "currentLevelName": nil, "taskProgress": nil}
		if kind == "pending" {
			entry["canApprove"] = canApprove
		}
		out = append(out, entry)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if kind == "apprentices" {
		for _, v := range out {
			if err = a.fillMentorProgress(r, v, true); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
func (a *App) fillMentorProgress(r *http.Request, v map[string]any, includeTasks bool) error {
	user := v["userId"].(int64)
	rows, err := a.DB.QueryContext(r.Context(), `SELECT l.id,l.name,CASE WHEN p.status='COMPLETED' THEN 1 ELSE 0 END FROM levels l LEFT JOIN user_level_progress p ON p.level_id=l.id AND p.user_id=? ORDER BY l.map_id,l.order_index,l.id`, user)
	if err != nil {
		return err
	}
	i, done := 0, 0
	currentFound := false
	var lastID int64
	var lastName string
	for rows.Next() {
		var id int64
		var name string
		var complete int
		if err = rows.Scan(&id, &name, &complete); err != nil {
			rows.Close()
			return err
		}
		lastID, lastName = id, name
		if complete == 1 {
			done++
		} else if !currentFound {
			v["currentLevelId"] = id
			v["currentLevelName"] = name
			v["currentLevelIndex"] = i
			currentFound = true
		}
		i++
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	v["totalLevels"] = i
	v["completedLevels"] = done
	if !currentFound && i > 0 {
		v["currentLevelId"] = lastID
		v["currentLevelName"] = lastName
		v["currentLevelIndex"] = i - 1
	}
	if includeTasks {
		rows, err = a.DB.QueryContext(r.Context(), `SELECT t.level_id,t.id,EXISTS(SELECT 1 FROM user_task_submissions s WHERE s.task_id=t.id AND s.user_id=? AND s.status='APPROVED') FROM tasks t`, user)
		if err != nil {
			return err
		}
		tasks := []any{}
		for rows.Next() {
			var lid sql.NullInt64
			var tid int64
			var done bool
			if err = rows.Scan(&lid, &tid, &done); err != nil {
				rows.Close()
				return err
			}
			tasks = append(tasks, map[string]any{"levelId": nullInt(lid), "taskId": tid, "completed": done})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		v["taskProgress"] = tasks
	}
	return nil
}
func (a *App) mentorshipMy(w http.ResponseWriter, r *http.Request, user int64) {
	v := map[string]any{}
	for _, k := range []string{"mentors", "apprentices", "pending", "sentRequests"} {
		list, err := a.relationList(r, user, k)
		if err != nil {
			fail(w)
			return
		}
		v[k] = list
	}
	ok(w, v)
}
func (a *App) mentorshipApprentices(w http.ResponseWriter, r *http.Request, user int64) {
	v, err := a.relationList(r, user, "apprentices")
	if err != nil {
		fail(w)
		return
	}
	ok(w, v)
}
func (a *App) mentorshipFellows(w http.ResponseWriter, r *http.Request, user int64) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT DISTINCT u.id,u.username,u.name,u.avatar,COALESCE(u.level,0) FROM mentorships mine JOIN mentorships fellow ON fellow.mentor_id=mine.mentor_id AND fellow.status='ACTIVE' JOIN users u ON u.id=fellow.apprentice_id WHERE mine.apprentice_id=? AND mine.status='ACTIVE' AND u.id<>?`, user, user)
	if err != nil {
		fail(w)
		return
	}
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var username string
		var name, avatar sql.NullString
		var level int
		if rows.Scan(&id, &username, &name, &avatar, &level) != nil {
			rows.Close()
			fail(w)
			return
		}
		out = append(out, map[string]any{"userId": id, "username": username, "name": nullString(name), "avatar": nullString(avatar), "level": level})
	}
	if rows.Err() != nil {
		rows.Close()
		fail(w)
		return
	}
	rows.Close()
	for _, v := range out {
		if a.fillMentorProgress(r, v, false) != nil {
			fail(w)
			return
		}
	}
	ok(w, out)
}
func (a *App) mentorshipSearch(w http.ResponseWriter, r *http.Request, user int64) {
	q := r.URL.Query().Get("q")
	if len(q) > 100 {
		bad(w, "查询过长")
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		SELECT mentor.id,mentor.username,mentor.name,mentor.role,mentor.total_xp,COALESCE(mentor.level,0)
		FROM users AS mentor JOIN users AS apprentice ON apprentice.id=?
		WHERE (`+eligibleMentorPair+`) AND NOT `+existingMentorPair+`
			AND (mentor.username LIKE ? OR mentor.name LIKE ?) ORDER BY mentor.id LIMIT 10`, user, "%"+q+"%", "%"+q+"%")
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var id int64
		var username, role string
		var name sql.NullString
		var xp, level int
		if rows.Scan(&id, &username, &name, &role, &xp, &level) != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": id, "username": username, "name": nullString(name), "role": role, "totalXp": xp, "level": level})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, out)
}
