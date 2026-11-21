package api

import (
	"reflect"
	"testing"
)

func TestTaskSubmissionsLimitPreservesHistoryAndOwnership(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, `INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 2,'another-student',password,'EXTERNAL',0,0 FROM users WHERE id=1;
		INSERT INTO user_task_submissions(id,user_id,task_id,submission_data,status,submitted_at,ai_review_count)
		VALUES (21,1,1,'first attempt','REJECTED','2026-01-01',0),
		       (22,1,1,'second attempt','PENDING','2026-01-02',0),
		       (23,1,1,'latest attempt','PENDING','2026-01-02',0),
		       (24,2,1,'another student','PENDING','2026-01-03',0),
		       (25,1,2,'another task','PENDING','2026-01-03',0)`)
	token := loginToken(t, h)
	type attempt struct {
		ID             int
		SubmissionData string
	}
	const history = "/api/tasks/1/submissions"
	var got []attempt
	learningResponse(t, h, "GET", history+"?limit=1", "", token, &got)
	if !reflect.DeepEqual(got, []attempt{{23, "latest attempt"}}) {
		t.Fatalf("editor prefill: %+v", got)
	}
	learningResponse(t, h, "GET", history, "", token, &got)
	if !reflect.DeepEqual(got, []attempt{{23, "latest attempt"}, {22, "second attempt"}, {21, "first attempt"}}) {
		t.Fatalf("complete history: %+v", got)
	}
	if w := call(h, "GET", history+"?limit=0", "", token); w.Code != 400 {
		t.Fatalf("expected invalid limit to fail, got %d: %s", w.Code, w.Body)
	}
}
