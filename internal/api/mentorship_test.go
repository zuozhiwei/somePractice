package api

import (
	"encoding/json"
	"testing"
)

func TestMentorshipRequiresStrictlyHigherRank(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, `UPDATE users SET role='INTERNAL',level=5 WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 2,'peer-higher',password,'TEACHER',10,6 FROM users WHERE id=1
		UNION ALL SELECT 3,'peer-equal',password,'INTERNAL',30,5 FROM users WHERE id=1
		UNION ALL SELECT 4,'peer-lower',password,'TEACHER',20,4 FROM users WHERE id=1
		UNION ALL SELECT 5,'peer-external',password,'EXTERNAL',40,10 FROM users WHERE id=1`)
	student := loginToken(t, h)
	var board struct {
		Content []struct {
			UserID           int
			CanRequestMentor bool
		}
	}
	learningResponse(t, h, "GET", "/api/leaderboard/global", "", student, &board)
	if len(board.Content) != 5 || board.Content[0].UserID != 5 {
		t.Fatalf("leaderboard order changed: %+v", board.Content)
	}
	for _, peer := range board.Content {
		if peer.CanRequestMentor != (peer.UserID == 2) {
			t.Fatalf("unexpected mentorship action for user %d: %v", peer.UserID, peer.CanRequestMentor)
		}
	}
	var candidates []struct{ ID int }
	searchPath := "/api/mentorship/search-users?q=peer"
	learningResponse(t, h, "GET", searchPath, "", student, &candidates)
	if len(candidates) != 1 || candidates[0].ID != 2 {
		t.Fatalf("wrong search candidates: %+v", candidates)
	}
	for _, body := range []string{`{"mentorId":3}`, `{"mentorId":4}`, `{"mentorId":5}`} {
		if w := call(h, "POST", "/api/mentorship/request", body, student); w.Code != 400 {
			t.Fatalf("invalid mentor request accepted for %s: %d %s", body, w.Code, w.Body)
		}
	}
	learningResponse(t, h, "POST", "/api/mentorship/request", `{"mentorId":2}`, student, nil)
	learningResponse(t, h, "GET", searchPath, "", student, &candidates)
	if len(candidates) != 0 {
		t.Fatalf("pending mentor still listed: %+v", candidates)
	}

	var mentor struct{ Token string }
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"peer-higher","password":"correct-password"}`, "", &mentor)
	var requests struct {
		Pending []struct{ CanApprove bool }
	}
	checkPending := func(eligible bool) {
		t.Helper()
		learningResponse(t, h, "GET", "/api/mentorship/my", "", mentor.Token, &requests)
		if len(requests.Pending) != 1 || requests.Pending[0].CanApprove != eligible {
			t.Fatalf("pending request eligibility: %+v", requests.Pending)
		}
	}
	checkPending(true)
	for _, rank := range []int{5, 4} {
		learningSQL(t, db, "UPDATE users SET level=? WHERE id=2", rank)
		checkPending(false)
		if w := call(h, "PUT", "/api/mentorship/1/approve", "", mentor.Token); w.Code != 400 {
			t.Fatalf("accepted request with mentor rank %d: %d %s", rank, w.Code, w.Body)
		}
	}
	learningSQL(t, db, "UPDATE users SET level=6 WHERE id=2")
	learningResponse(t, h, "PUT", "/api/mentorship/1/approve", "", mentor.Token, nil)
}

func TestMentorshipEndsOnlyAfterStrictOvertake(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, `UPDATE users SET role='INTERNAL' WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 2,'mentor',password,'TEACHER',0,35 FROM users WHERE id=1
		UNION ALL SELECT 3,'admin',password,'SUPERADMIN',0,100 FROM users WHERE id=1
		UNION ALL SELECT 4,'another-mentor',password,'INTERNAL',0,80 FROM users WHERE id=1;
		INSERT INTO mentorships(id,mentor_id,apprentice_id,status) VALUES(1,2,1,'ACTIVE'),(2,4,1,'ACTIVE')`)
	student := loginToken(t, h)
	var mentor, admin struct{ Token string }
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"mentor","password":"correct-password"}`, "", &mentor)
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"admin","password":"correct-password"}`, "", &admin)
	status := func(id int, want string) {
		t.Helper()
		var got string
		if err := db.QueryRow("SELECT status FROM mentorships WHERE id=?", id).Scan(&got); err != nil || got != want {
			t.Fatalf("relationship %d status = %q (%v), want %q", id, got, err, want)
		}
	}
	learningResponse(t, h, "POST", "/api/tasks/1/submit", `{"submissionData":"first task"}`, student, nil)
	learningResponse(t, h, "PUT", "/api/submissions/1/review", `{"status":"APPROVED"}`, mentor.Token, nil)
	status(1, "ACTIVE")
	learningResponse(t, h, "POST", "/api/tasks/2/submit", `{"submissionData":"second task"}`, student, nil)
	learningResponse(t, h, "PUT", "/api/submissions/2/review", `{"status":"APPROVED"}`, mentor.Token, nil)
	status(1, "ENDED")
	status(2, "ACTIVE")
	learningResponse(t, h, "PUT", "/api/admin/users/4/level?level=70", "", admin.Token, nil)
	status(2, "ACTIVE")
	learningResponse(t, h, "PUT", "/api/admin/users/4/level?level=69", "", admin.Token, nil)
	status(2, "ENDED")
}

func TestMentorshipInvitationRequiresHigherRankAtRegistration(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, "UPDATE users SET role='INTERNAL',level=1 WHERE id=1")
	mentor := loginToken(t, h)
	var invitation string
	learningResponse(t, h, "GET", "/api/mentorship/invite-link", "", mentor, &invitation)
	learningSQL(t, db, "UPDATE users SET level=0 WHERE id=1")
	if w := call(h, "GET", "/api/mentorship/invite-link", "", mentor); w.Code != 400 {
		t.Fatalf("rank-zero invitation issued: %d %s", w.Code, w.Body)
	}
	payload, err := json.Marshal(map[string]string{
		"username": "new-student", "password": "test-password", "role": "INTERNAL", "inviteToken": invitation,
	})
	if err != nil {
		t.Fatal(err)
	}
	if w := call(h, "POST", "/api/auth/register", string(payload), ""); w.Code != 400 {
		t.Fatalf("invitation accepted after rank dropped: %d %s", w.Code, w.Body)
	}
}
