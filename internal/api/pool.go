package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const submissionSelect = `SELECT s.id,s.user_id,s.task_id,COALESCE(u.username,''),u.name,COALESCE(t.title,''),s.status,s.submission_data,s.ai_feedback,s.review_comment,s.submitted_at,s.reviewed_at FROM user_task_submissions s LEFT JOIN users u ON u.id=s.user_id LEFT JOIN tasks t ON t.id=s.task_id `

func (a *App) submissionRow(rows *sql.Rows) (map[string]any, error) {
	var id, uid, tid int64
	var username, title string
	var name, status, data, feedback, comment, submitted, reviewed sql.NullString
	if err := rows.Scan(&id, &uid, &tid, &username, &name, &title, &status, &data, &feedback, &comment, &submitted, &reviewed); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "userId": uid, "taskId": tid, "username": username, "name": nullString(name), "taskTitle": title, "status": nullString(status), "submissionData": signedContent(data), "aiFeedback": nullString(feedback), "reviewComment": nullString(comment), "submittedAt": a.timestamp(submitted), "reviewedAt": a.timestamp(reviewed)}, nil
}
func (a *App) submissionDetail(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "提交编号无效")
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), submissionSelect+`WHERE s.id=? AND (s.user_id=? OR EXISTS(SELECT 1 FROM mentorships m WHERE m.mentor_id=? AND m.apprentice_id=s.user_id AND m.status='ACTIVE') OR EXISTS(SELECT 1 FROM users reviewer JOIN users student ON student.id=s.user_id WHERE reviewer.id=? AND student.role='EXTERNAL' AND reviewer.role IN ('INTERNAL','TEACHER') AND COALESCE(reviewer.level,0)>COALESCE(student.level,0)) OR s.reviewer_id=?)`, id, user, user, user, user)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	if !rows.Next() {
		reply(w, 403, nil, "提交不存在或无权查看")
		return
	}
	v, err := a.submissionRow(rows)
	if err != nil {
		fail(w)
		return
	}
	ok(w, v)
}
func pagination(r *http.Request) (int, int, error) {
	page, size := 0, 10
	var e error
	if v := r.URL.Query().Get("page"); v != "" {
		page, e = strconv.Atoi(v)
		if e != nil || page < 0 || page > 1000000 {
			return 0, 0, fmt.Errorf("invalid page")
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		size, e = strconv.Atoi(v)
		if e != nil || size < 1 || size > 100 {
			return 0, 0, fmt.Errorf("invalid size")
		}
	}
	return page, size, nil
}
func (a *App) submissionPool(w http.ResponseWriter, r *http.Request, user int64, kind string) {
	page, size, err := pagination(r)
	if err != nil {
		bad(w, "分页参数无效")
		return
	}
	var where string
	var args []any
	if kind == "apprentice" {
		where = `EXISTS(SELECT 1 FROM mentorships m WHERE m.mentor_id=? AND m.apprentice_id=s.user_id AND m.status='ACTIVE') AND s.id=(SELECT x.id FROM user_task_submissions x WHERE x.user_id=s.user_id AND x.task_id=s.task_id ORDER BY x.submitted_at DESC,x.id DESC LIMIT 1)`
		args = []any{user}
	} else {
		where = `EXISTS(SELECT 1 FROM users reviewer JOIN users student ON student.id=s.user_id WHERE reviewer.id=? AND reviewer.role IN ('INTERNAL','TEACHER') AND student.role='EXTERNAL' AND ((s.status='PENDING' AND COALESCE(reviewer.level,0)>COALESCE(student.level,0)) OR s.reviewer_id=reviewer.id))`
		args = []any{user}
	}
	var total int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions s WHERE `+where, args...).Scan(&total); err != nil {
		fail(w)
		return
	}
	args = append(args, size, page*size)
	rows, err := a.DB.QueryContext(r.Context(), submissionSelect+`WHERE `+where+` ORDER BY s.submitted_at DESC,s.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		v, e := a.submissionRow(rows)
		if e != nil {
			fail(w)
			return
		}
		delete(v, "submissionData")
		out = append(out, v)
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"content": out, "total": total})
}
func (a *App) apprenticeSubmissions(w http.ResponseWriter, r *http.Request, user int64) {
	a.submissionPool(w, r, user, "apprentice")
}
func (a *App) externalSubmissions(w http.ResponseWriter, r *http.Request, user int64) {
	a.submissionPool(w, r, user, "external")
}
func (a *App) aiReview(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "提交编号无效")
		return
	}
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		reply(w, 503, nil, "AI 审核未配置")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	var task int64
	var data, desc string
	var criteria sql.NullString
	var enabled, used int
	err = tx.QueryRowContext(r.Context(), `SELECT s.task_id,COALESCE(s.submission_data,''),COALESCE(t.description,''),t.acceptance_criteria,t.ai_review_enabled,(SELECT COALESCE(sum(x.ai_review_count),0) FROM user_task_submissions x WHERE x.user_id=s.user_id AND x.task_id=s.task_id) FROM user_task_submissions s JOIN tasks t ON t.id=s.task_id WHERE s.id=? AND s.user_id=? AND s.status='PENDING'`, id, user).Scan(&task, &data, &desc, &criteria, &enabled, &used)
	if err != nil {
		tx.Rollback()
		bad(w, "提交不存在或不可审核")
		return
	}
	if enabled == 0 || used >= 3 {
		tx.Rollback()
		bad(w, "未开启 AI 审核或已超过次数")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE user_task_submissions SET ai_review_count=ai_review_count+1 WHERE id=?`, id); err != nil {
		tx.Rollback()
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	// A failed network call or malformed response must not consume an attempt.
	keepAttempt := false
	defer func() {
		if !keepAttempt {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = a.DB.ExecContext(ctx, `UPDATE user_task_submissions SET ai_review_count=MAX(ai_review_count-1,0) WHERE id=? AND user_id=?`, id, user)
		}
	}()
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	body, _ := json.Marshal(map[string]any{"model": model, "messages": []map[string]string{{"role": "system", "content": "你是编程任务评估助手。只返回 JSON：{\"pass\":true或false,\"feedback\":\"评价\",\"score\":0到100}"}, {"role": "user", "content": "任务描述：" + desc + "\n验收标准：" + criteria.String + "\n学生提交：" + data}}})
	endpoint := os.Getenv("DEEPSEEK_API_URL")
	if endpoint == "" {
		endpoint = "https://api.deepseek.com/v1/chat/completions"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		fail(w)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		reply(w, 502, nil, "AI 服务请求失败")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		reply(w, 502, nil, "AI 服务暂时不可用")
		return
	}
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&apiResp) != nil || len(apiResp.Choices) == 0 {
		reply(w, 502, nil, "AI 返回格式错误")
		return
	}
	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	var result struct {
		Pass     bool   `json:"pass"`
		Feedback string `json:"feedback"`
		Score    int    `json:"score"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(content)), &result) != nil {
		reply(w, 502, nil, "AI 返回格式错误")
		return
	}
	feedback := fmt.Sprintf("AI评分: %d/100\n%s", result.Score, result.Feedback)
	tx, err = a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var current string
	var xp int
	var lid sql.NullInt64
	if err = tx.QueryRowContext(r.Context(), `SELECT s.status,t.xp_reward,t.level_id FROM user_task_submissions s JOIN tasks t ON t.id=s.task_id WHERE s.id=? AND s.user_id=?`, id, user).Scan(&current, &xp, &lid); err != nil || current != "PENDING" {
		bad(w, "提交状态已发生变化")
		return
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	if result.Pass {
		var granted int
		if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions WHERE user_id=? AND task_id=? AND xp_granted=1`, user, task).Scan(&granted); err != nil {
			fail(w)
			return
		}
		_, err = tx.ExecContext(r.Context(), `UPDATE user_task_submissions SET status='APPROVED',ai_feedback=?,review_comment='AI自动审核通过',reviewed_at=?,xp_granted=CASE WHEN ?=0 THEN 1 ELSE xp_granted END WHERE id=?`, feedback, now, granted, id)
		if err == nil && granted == 0 {
			_, err = tx.ExecContext(r.Context(), `UPDATE users SET total_xp=total_xp+? WHERE id=?`, xp, user)
		}
		if err == nil && lid.Valid {
			err = updateProgress(r.Context(), tx, user, lid.Int64, now)
		}
		if err == nil && lid.Valid {
			err = completeLevel(r.Context(), tx, user, lid.Int64, now)
		}
	} else {
		_, err = tx.ExecContext(r.Context(), `UPDATE user_task_submissions SET ai_feedback=? WHERE id=?`, feedback, id)
	}
	if err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	keepAttempt = true
	ok(w, map[string]any{"id": id, "status": map[bool]string{true: "APPROVED", false: "PENDING"}[result.Pass], "aiFeedback": feedback})
}
