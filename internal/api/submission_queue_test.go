package api

import (
	"slices"
	"testing"
)

func TestSubmissionQueuesFilterAndPreserveReviewHistory(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, `
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 2,'mentor',password,'INTERNAL',0,50 FROM users WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 3,'apprentice',password,'INTERNAL',0,1 FROM users WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 4,'teacher',password,'TEACHER',0,30 FROM users WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 5,'admin',password,'SUPERADMIN',0,70 FROM users WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 6,'advanced-external',password,'EXTERNAL',0,60 FROM users WHERE id=1;
		INSERT INTO mentorships(mentor_id,apprentice_id,status) VALUES(2,3,'ACTIVE');
		INSERT INTO user_task_submissions(id,user_id,task_id,status,submitted_at,reviewed_at,reviewer_id,submission_data,ai_feedback,review_comment,ai_review_count) VALUES
			(1,3,1,'PENDING','2026-11-01 09:00:00',NULL,NULL,'superseded',NULL,NULL,0),
			(2,3,1,'REJECTED','2026-11-02 09:00:00','2026-11-03 09:00:00',2,'old attempt',NULL,'needs updates',0),
			(3,3,1,'AI_REVIEWED','2026-11-06 09:00:00',NULL,NULL,'new attempt','AI评分: 82/100',NULL,1),
			(4,3,2,'PENDING','2026-11-05 09:00:00',NULL,NULL,'second task',NULL,NULL,0),
			(5,3,2,'REJECTED','2026-11-01 09:00:00','2026-11-04 09:00:00',2,'second old attempt',NULL,'retry',0),
			(6,1,1,'PENDING','2026-11-06 09:00:00',NULL,NULL,'external pending',NULL,NULL,0),
			(7,1,2,'AI_REVIEWED','2026-11-07 09:00:00',NULL,NULL,'external AI',NULL,NULL,1),
			(8,1,1,'REJECTED','2026-11-02 09:00:00','2026-11-04 09:00:00',2,'external rejected',NULL,'revise',0),
			(9,1,2,'APPROVED','2026-11-01 09:00:00','2026-11-08 09:00:00',4,'external complete',NULL,'ok',0),
			(10,6,1,'PENDING','2026-11-10 09:00:00',NULL,NULL,'higher rank',NULL,NULL,0),
			(11,5,1,'PENDING','2026-11-12 09:00:00',NULL,NULL,'admin own',NULL,NULL,0)`)

	login := func(username string) string {
		var auth struct{ Token string }
		learningResponse(t, h, "POST", "/api/auth/login", `{"username":"`+username+`","password":"correct-password"}`, "", &auth)
		return auth.Token
	}
	mentor := login("mentor")
	apprentice := login("apprentice")
	teacher := login("teacher")
	admin := login("admin")
	external := loginToken(t, h)
	assertPage := func(path, token string, total int, ids ...int) {
		t.Helper()
		var result struct {
			Content []map[string]any
			Total   int
		}
		learningResponse(t, h, "GET", path, "", token, &result)
		actual := make([]int, 0, len(result.Content))
		for _, row := range result.Content {
			actual = append(actual, int(row["id"].(float64)))
			for _, field := range []string{"submissionData", "aiFeedback", "reviewComment"} {
				if _, exists := row[field]; exists {
					t.Errorf("queue leaked %s in %+v", field, row)
				}
			}
		}
		if result.Total != total || result.Content == nil || !slices.Equal(actual, ids) {
			t.Fatalf("%s got ids %v total %d; want %v total %d", path, actual, result.Total, ids, total)
		}
	}
	const apprenticePath = "/api/submissions/apprentice"
	const externalPath = "/api/submissions/external"
	const adminPath = "/api/admin/submissions"
	assertPage(apprenticePath+"?queue=pending&size=1", mentor, 2, 3)
	assertPage(apprenticePath+"?queue=pending&page=1&size=1", mentor, 2, 4)
	assertPage(apprenticePath+"?queue=pending&page=2&size=1", mentor, 2)
	assertPage(apprenticePath+"?queue=reviewed", mentor, 2, 5, 2)
	assertPage(apprenticePath+"?queue=pending", teacher, 0)
	assertPage(externalPath+"?queue=pending", mentor, 2, 7, 6)
	assertPage(externalPath+"?queue=reviewed", mentor, 1, 8)
	assertPage(externalPath+"?queue=reviewed", teacher, 1, 9)
	assertPage(externalPath+"?queue=pending", external, 0)
	assertPage(adminPath+"?queue=pending", teacher, 2, 7, 6)
	assertPage(adminPath+"?queue=pending&size=2", admin, 6, 10, 7)
	assertPage(adminPath+"?queue=reviewed", teacher, 2, 9, 8)
	if w := call(h, "GET", adminPath+"?queue=pending", "", mentor); w.Code != 403 {
		t.Fatalf("non-staff admin queue returned %d", w.Code)
	}
	if w := call(h, "GET", apprenticePath+"?queue=invalid", "", mentor); w.Code != 400 {
		t.Fatalf("invalid queue returned %d", w.Code)
	}

	var legacy struct{ Content []map[string]any }
	learningResponse(t, h, "GET", apprenticePath, "", mentor, &legacy)
	if len(legacy.Content) != 2 || legacy.Content[0]["aiFeedback"] != "AI评分: 82/100" {
		t.Fatalf("legacy apprentice summary changed: %+v", legacy)
	}
	var detail struct{ SubmissionData, AIFeedback, ReviewComment string }
	learningResponse(t, h, "GET", "/api/submissions/3", "", mentor, &detail)
	if detail.SubmissionData != "new attempt" || detail.AIFeedback != "AI评分: 82/100" {
		t.Fatalf("detail lost work or feedback: %+v", detail)
	}
	if w := call(h, "GET", "/api/submissions/3", "", teacher); w.Code != 403 {
		t.Fatalf("unassigned teacher detail returned %d", w.Code)
	}

	learningResponse(t, h, "PUT", "/api/submissions/3/review", `{"status":"APPROVED","reviewComment":"new approval"}`, mentor, nil)
	assertPage(apprenticePath+"?queue=pending", mentor, 1, 4)
	assertPage(apprenticePath+"?queue=reviewed", mentor, 3, 3, 5, 2)
	learningResponse(t, h, "GET", "/api/submissions/3", "", mentor, &detail)
	if detail.ReviewComment != "new approval" || detail.AIFeedback != "AI评分: 82/100" {
		t.Fatalf("approval lost review history: %+v", detail)
	}
	learningResponse(t, h, "PUT", "/api/submissions/4/review", `{"status":"REJECTED"}`, mentor, nil)
	assertPage(apprenticePath+"?queue=pending", mentor, 0)
	var newAttempt struct{ ID int }
	learningResponse(t, h, "POST", "/api/tasks/2/submit", `{"submissionData":"corrected work"}`, apprentice, &newAttempt)
	assertPage(apprenticePath+"?queue=pending", mentor, 1, newAttempt.ID)
	assertPage(apprenticePath+"?queue=reviewed", mentor, 4, 4, 3, 5, 2)
}
