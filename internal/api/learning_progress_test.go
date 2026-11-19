package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"skilltree-go/internal/curriculum"
)

func learningSQL(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatal(err)
	}
}

func learningResponse(t *testing.T, h http.Handler, method, path, body, token string, target any) {
	t.Helper()
	w := call(h, method, path, body, token)
	if w.Code != 200 {
		t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
	}
	if target != nil {
		var env struct{ Data json.RawMessage }
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(env.Data, target); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
}

func TestRecommendationsFollowPublishedBranches(t *testing.T) {
	db, h := testApp(t)
	token := loginToken(t, h)
	learningSQL(t, db, "DELETE FROM tutorials; DELETE FROM tasks; DELETE FROM levels; DELETE FROM maps")
	pack, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pack.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	// In the published course, levels 27, 28 and 29 branch off 26.
	// The next milestone (30) requires the two branches 28 and 29.
	learningSQL(t, db, `INSERT INTO user_level_progress(user_id,level_id,status,progress,xp_earned)
		SELECT 1,id,'COMPLETED',100,0 FROM levels WHERE id<=26;
		INSERT INTO user_task_submissions(user_id,task_id,status,ai_review_count)
		SELECT 1,id,CASE WHEN level_id=27 THEN 'AI_REVIEWED' ELSE 'PENDING' END,0 FROM tasks WHERE level_id IN (27,28)`)
	check := func(path, state string, level int) {
		t.Helper()
		var got struct {
			State     string
			Level     struct{ ID int }
			Task      struct{ ID int }
			Tutorials []map[string]any
		}
		learningResponse(t, h, "GET", path, "", token, &got)
		if got.State != state || got.Level.ID != level || (level != 0 && (got.Task.ID == 0 || len(got.Tutorials) == 0)) {
			t.Fatalf("wrong mission: %+v", got)
		}
	}
	const mission = "/api/adventure/mission"
	check(mission, "AVAILABLE", 29)
	check(mission+"?chapterId=5", "AVAILABLE", 29)
	check(mission+"?chapterId=6", "BLOCKED", 0)
	learningSQL(t, db, `INSERT INTO user_task_submissions(user_id,task_id,status,ai_review_count) SELECT 1,id,'PENDING',0 FROM tasks WHERE level_id=29`)
	check(mission+"?chapterId=5", "WAITING_REVIEW", 27)
	learningSQL(t, db, `INSERT INTO user_level_progress(user_id,level_id,status,progress,xp_earned) VALUES (1,28,'COMPLETED',100,0),(1,29,'COMPLETED',100,0)`)
	check(mission, "AVAILABLE", 30)
}

func TestTaskApprovalAndLatestFeedbackAcrossStudentViews(t *testing.T) {
	db, h := testApp(t)
	token := loginToken(t, h)
	learningSQL(t, db, `INSERT INTO users(id,username,password,role,total_xp,level) SELECT 2,'other',password,'EXTERNAL',0,0 FROM users WHERE id=1;
		INSERT INTO user_task_submissions(id,user_id,task_id,status,submitted_at,reviewed_at,ai_feedback,review_comment,ai_review_count)
		VALUES(1,1,1,'APPROVED','2025-01-01','2025-01-01','first','passed',0),
		      (2,1,1,'REJECTED','2025-01-02','2025-01-05',NULL,'most recent review',0),
		      (3,1,1,'REJECTED','2025-01-03','2025-01-04','recent AI','earlier review',0),
		      (4,1,1,'PENDING','2025-01-06',NULL,'  ',NULL,0),
		      (5,2,1,'REJECTED','2025-01-07','2025-01-08','other user','other review',0)`)
	type taskState struct{ SubmissionStatus, AiFeedback, ReviewComment string }
	var detail struct{ Tasks []taskState }
	var tasks []taskState
	learningResponse(t, h, "GET", "/api/levels/1", "", token, &detail)
	learningResponse(t, h, "GET", "/api/tasks?levelId=1", "", token, &tasks)
	expected := []taskState{{"APPROVED", "recent AI", "most recent review"}}
	if !reflect.DeepEqual(detail.Tasks, expected) || !reflect.DeepEqual(tasks, expected) {
		t.Fatalf("level=%+v tasks=%+v", detail.Tasks, tasks)
	}
	if w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"resubmit"}`, token); w.Code != 400 {
		t.Fatalf("resubmitting completed work: %d", w.Code)
	}
	learningSQL(t, db, `UPDATE user_task_submissions SET status='REJECTED' WHERE id=1`)
	var mission struct {
		State string
		Task  taskState
	}
	learningResponse(t, h, "GET", "/api/adventure/mission", "", token, &mission)
	if mission.State != "WAITING_REVIEW" || mission.Task != (taskState{"PENDING", "recent AI", "most recent review"}) {
		t.Fatalf("mission lost feedback: %+v", mission)
	}
}

func TestLegacyAIReviewedCanBeReviewedAndSupplemented(t *testing.T) {
	db, h := testApp(t)
	student := loginToken(t, h)
	learningSQL(t, db, `INSERT INTO users(id,username,password,role,total_xp,level) SELECT 2,'reviewer',password,'TEACHER',0,20 FROM users WHERE id=1;
		INSERT INTO user_task_submissions(id,user_id,task_id,status,ai_review_count) VALUES(1,1,1,'AI_REVIEWED',0)`)
	var auth struct{ Token string }
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"reviewer","password":"correct-password"}`, "", &auth)
	var pool struct {
		Total   int
		Content []struct {
			ID     int
			Status string
		}
	}
	learningResponse(t, h, "GET", "/api/submissions/external", "", auth.Token, &pool)
	if pool.Total != 1 || len(pool.Content) != 1 || pool.Content[0].Status != "AI_REVIEWED" {
		t.Fatalf("missing legacy submission in pool: %+v", pool)
	}
	learningResponse(t, h, "POST", "/api/tasks/1/submit", `{"submissionData":"supplement"}`, student, nil)
	learningResponse(t, h, "PUT", "/api/submissions/1/review", `{"status":"APPROVED","reviewComment":"passed"}`, auth.Token, nil)
	// The reviewer can repeat their decision after the student outranks them.
	learningResponse(t, h, "PUT", "/api/submissions/1/review", `{"status":"APPROVED","reviewComment":"passed"}`, auth.Token, nil)
	var detail struct {
		Tasks []struct{ SubmissionStatus string }
	}
	learningResponse(t, h, "GET", "/api/levels/1", "", student, &detail)
	if len(detail.Tasks) != 1 || detail.Tasks[0].SubmissionStatus != "APPROVED" {
		t.Fatalf("late review lost: %+v", detail.Tasks)
	}
}

func TestLevelScalesWithPlayableCourseAndKeepsEarnedRank(t *testing.T) {
	db, h := testApp(t)
	token := loginToken(t, h)
	learningSQL(t, db, `WITH RECURSIVE ids(n) AS (VALUES(3) UNION ALL SELECT n+1 FROM ids WHERE n<47)
		INSERT INTO levels(id,is_builtin,map_id,name,order_index,tier,xp_reward,prerequisite_level_ids)
			SELECT n,1,1,'level-'||n,n,1,1,'[]' FROM ids;
		INSERT INTO tasks(id,title,level_id,xp_reward,ai_review_enabled,is_builtin) SELECT id,name,id,1,0,1 FROM levels WHERE id BETWEEN 3 AND 46;
		INSERT INTO user_level_progress(user_id,level_id,status,progress,xp_earned) SELECT 1,id,'COMPLETED',100,0 FROM levels WHERE id<=22;
		INSERT INTO user_task_submissions(user_id,task_id,status,ai_review_count) VALUES (1,23,'APPROVED',0),(1,24,'APPROVED',0)`)
	learningResponse(t, h, "PUT", "/api/levels/23/complete", "", token, nil)
	var grade int
	if err := db.QueryRow("SELECT level FROM users WHERE id=1").Scan(&grade); err != nil || grade != 35 {
		t.Fatalf("23 of 46 playable levels: %d %v", grade, err)
	}
	learningSQL(t, db, `UPDATE users SET level=65 WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level) SELECT 2,'mentor',password,'TEACHER',0,70 FROM users WHERE id=1;
		INSERT INTO mentorships(mentor_id,apprentice_id,status) VALUES(2,1,'ACTIVE')`)
	learningResponse(t, h, "PUT", "/api/levels/24/complete", "", token, nil)
	var relationship string
	if err := db.QueryRow("SELECT u.level,m.status FROM users u JOIN mentorships m ON m.apprentice_id=u.id WHERE u.id=1").Scan(&grade, &relationship); err != nil || grade != 65 || relationship != "ACTIVE" {
		t.Fatalf("legacy rank %d, relationship %s: %v", grade, relationship, err)
	}
}

func TestAtlasVisitsArePersonalAndIdempotent(t *testing.T) {
	db, h := testApp(t)
	first := loginToken(t, h)
	learningSQL(t, db, `INSERT INTO users(id,username,password,role,total_xp,level) SELECT 2,'other',password,'EXTERNAL',0,0 FROM users WHERE id=1`)
	var auth struct{ Token string }
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"other","password":"correct-password"}`, "", &auth)
	if w := call(h, "POST", "/api/maps/2/visit", "", ""); w.Code != 401 {
		t.Fatalf("anonymous visit: %d", w.Code)
	}
	if w := call(h, "POST", "/api/maps/999/visit", "", first); w.Code != 404 {
		t.Fatalf("missing chapter: %d", w.Code)
	}
	for range 3 {
		learningResponse(t, h, "POST", "/api/maps/2/visit", "", first, nil)
	}
	learningResponse(t, h, "POST", "/api/maps/1/visit", "", auth.Token, nil)
	for token, expected := range map[string][]int{first: {2}, auth.Token: {1}} {
		var atlas struct{ VisitedChapterIDs []int }
		learningResponse(t, h, "GET", "/api/maps/atlas", "", token, &atlas)
		if !reflect.DeepEqual(atlas.VisitedChapterIDs, expected) {
			t.Fatalf("visits %+v; want %+v", atlas.VisitedChapterIDs, expected)
		}
	}
	var visits, progress int
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM atlas_visits), (SELECT count(*) FROM user_level_progress)`).Scan(&visits, &progress); err != nil || visits != 2 || progress != 0 {
		t.Fatalf("visits=%d learning progress=%d: %v", visits, progress, err)
	}
}
