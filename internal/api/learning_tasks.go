package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

type learningTask struct {
	task
	Status                  string
	Feedback, ReviewComment sql.NullString
}

func (t learningTask) response() map[string]any {
	return map[string]any{
		"id": t.ID, "title": t.Title, "description": nullString(t.Description),
		"acceptanceCriteria": criteria(t.task), "xpReward": t.XP, "sortOrder": nullInt(t.Order),
		"submissionStatus": t.Status, "aiFeedback": nullString(t.Feedback),
		"reviewComment": nullString(t.ReviewComment), "aiReviewEnabled": t.AI != 0,
	}
}

func awaitingReview(status string) bool {
	return status == "PENDING" || status == "AI_REVIEWED"
}

// The caller supplies accessible levels in curriculum order. One query reads only
// this student's history in those levels. Approval survives later submissions,
// while each feedback field keeps its most recent nonblank review.
func (a *App) learningTasks(ctx context.Context, user int64, levelIDs []int64, recommend bool) ([]learningTask, error) {
	ids, err := json.Marshal(levelIDs)
	if err != nil {
		return nil, err
	}
	query := `WITH scope AS (
		SELECT t.id, l.key AS position FROM json_each(?) l JOIN tasks t ON t.level_id=l.value
	), history AS (
		SELECT s.task_id, s.status,
		       max(s.status='APPROVED') OVER (PARTITION BY s.task_id) AS passed,
		       row_number() OVER (PARTITION BY s.task_id ORDER BY s.submitted_at DESC, s.id DESC) AS recent,
		       first_value(nullif(trim(s.ai_feedback),'')) OVER (PARTITION BY s.task_id
		         ORDER BY nullif(trim(s.ai_feedback),'') IS NULL, coalesce(s.reviewed_at,s.submitted_at) DESC, s.id DESC) AS feedback,
		       first_value(nullif(trim(s.review_comment),'')) OVER (PARTITION BY s.task_id
		         ORDER BY nullif(trim(s.review_comment),'') IS NULL, coalesce(s.reviewed_at,s.submitted_at) DESC, s.id DESC) AS comment
		FROM user_task_submissions s JOIN scope ON scope.id=s.task_id WHERE s.user_id=?
	), progress AS (
		SELECT scope.id, scope.position, h.feedback, h.comment,
		       CASE WHEN h.passed THEN 'APPROVED' ELSE coalesce(h.status,'NONE') END AS status
		FROM scope LEFT JOIN history h ON h.task_id=scope.id AND h.recent=1
	)
	SELECT t.id,t.title,t.description,t.acceptance_criteria,t.xp_reward,t.sort_order,
	       t.ai_review_enabled,t.is_builtin,t.created_by,t.created_at,t.level_id,p.status,p.feedback,p.comment
	FROM progress p JOIN tasks t ON t.id=p.id `
	if recommend {
		query += `WHERE p.status<>'APPROVED' ORDER BY p.status IN ('PENDING','AI_REVIEWED'), p.position,t.sort_order,t.id LIMIT 1`
	} else {
		query += `ORDER BY p.position,t.sort_order,t.id`
	}
	rows, err := a.DB.QueryContext(ctx, query, string(ids), user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []learningTask{}
	for rows.Next() {
		var t learningTask
		if err = rows.Scan(&t.ID, &t.Title, &t.Description, &t.Criteria, &t.XP, &t.Order,
			&t.AI, &t.Builtin, &t.CreatedBy, &t.CreatedAt, &t.LevelID, &t.Status, &t.Feedback, &t.ReviewComment); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (a *App) levelTutorials(ctx context.Context, levelID int64) ([]map[string]any, error) {
	rows, err := a.DB.QueryContext(ctx, `SELECT id,title,content,sort_order FROM tutorials WHERE level_id=? ORDER BY sort_order,id`, levelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var title string
		var content sql.NullString
		var order int
		if err = rows.Scan(&id, &title, &content, &order); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "content": nullString(content), "sortOrder": order})
	}
	return out, rows.Err()
}
