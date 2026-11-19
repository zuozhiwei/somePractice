package api

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"skilltree-go/internal/store"
)

func testApp(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	t.Setenv("SKILLTREE_JWT_SECRET", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 64))))
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(1,'alice',?,'EXTERNAL',0,0)`,
		`INSERT INTO maps(id,is_builtin,name,order_index) VALUES(1,1,'First',1),(2,1,'Second',2)`,
		`INSERT INTO levels(id,is_builtin,map_id,name,order_index,tier,xp_reward,prerequisite_level_ids) VALUES(1,1,1,'Start',1,1,10,'[]'),(2,1,2,'Locked',2,1,10,'[1]')`,
		`INSERT INTO tasks(id,title,level_id,xp_reward,ai_review_enabled,is_builtin,sort_order) VALUES(1,'Task one',1,5,0,1,1),(2,'Task two',2,5,0,1,1)`,
		`INSERT INTO tutorials(id,title,level_id,sort_order,is_builtin) VALUES(1,'Guide',1,1,1)`,
	}
	for i, s := range statements {
		if i == 0 {
			_, err = db.Exec(s, string(hash))
		} else {
			_, err = db.Exec(s)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	m := http.NewServeMux()
	New(db).Register(m)
	return db, m
}
func call(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	h.ServeHTTP(w, r)
	return w
}
func loginToken(t *testing.T, h http.Handler) string {
	t.Helper()
	w := call(h, "POST", "/api/auth/login", `{"username":"alice","password":"correct-password"}`, "")
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body)
	}
	var response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data.Token
}
func TestLoginAndLearningFlow(t *testing.T) {
	db, h := testApp(t)
	if w := call(h, "POST", "/api/auth/login", `{"username":"alice","password":"bad"}`, ""); w.Code != 400 {
		t.Fatalf("bad password: %d", w.Code)
	}
	if w := call(h, "GET", "/api/maps", "", ""); w.Code != 401 {
		t.Fatalf("anonymous: %d", w.Code)
	}
	token := loginToken(t, h)
	w := call(h, "GET", "/api/maps", "", token)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"status":"LOCKED"`)) {
		t.Fatalf("maps %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", "/api/levels/2", "", token)
	if w.Code != 400 {
		t.Fatalf("locked %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", "/api/levels/1", "", token)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"submissionStatus":"NONE"`)) {
		t.Fatalf("detail %d %s", w.Code, w.Body)
	}
	w = call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"my work"}`, token)
	if w.Code != 200 {
		t.Fatalf("submit %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", "/api/tasks/1/submissions", "", token)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("my work")) {
		t.Fatalf("history %d %s", w.Code, w.Body)
	}
	if _, err := db.Exec(`UPDATE user_task_submissions SET status='APPROVED' WHERE user_id=1 AND task_id=1`); err != nil {
		t.Fatal(err)
	}
	w = call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"again"}`, token)
	if w.Code != 400 {
		t.Fatalf("duplicate approved %d %s", w.Code, w.Body)
	}
	if _, err := db.Exec(`UPDATE user_level_progress SET status='COMPLETED',progress=100 WHERE user_id=1 AND level_id=1`); err != nil {
		t.Fatal(err)
	}
	w = call(h, "GET", "/api/levels/2", "", token)
	if w.Code != 200 {
		t.Fatalf("unlocked %d %s", w.Code, w.Body)
	}
}
func TestExpiredTokenRejected(t *testing.T) {
	_, h := testApp(t)
	token := loginToken(t, h)
	parts := strings.Split(token, ".")
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"alice","userId":1,"exp":1}`))
	w := call(h, "GET", "/api/auth/profile", "", strings.Join(parts, "."))
	if w.Code != 401 {
		t.Fatal(fmt.Sprintf("tampered token: %d", w.Code))
	}
}

func TestAdminRoleChangePreservesPassword(t *testing.T) {
	db, h := testApp(t)
	if _, err := db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 2,'operator',password,'SUPERADMIN',0,0 FROM users WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	login := call(h, "POST", "/api/auth/login", `{"username":"operator","password":"correct-password"}`, "")
	var response struct {
		Data struct {
			Token string
		}
	}
	if login.Code != 200 || json.Unmarshal(login.Body.Bytes(), &response) != nil || response.Data.Token == "" {
		t.Fatalf("superadmin login returned %d", login.Code)
	}
	if w := call(h, "PUT", "/api/admin/users/1/role?role=INTERNAL", "", response.Data.Token); w.Code != 200 {
		t.Fatalf("role update %d %s", w.Code, w.Body)
	}
	w := call(h, "POST", "/api/auth/login", `{"username":"alice","password":"correct-password"}`, "")
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"role":"INTERNAL"`)) {
		t.Fatalf("login after role update returned %d", w.Code)
	}
}

func TestStaffPermissionBoundaries(t *testing.T) {
	db, h := testApp(t)
	for id, role := range []string{"TEACHER", "SUPERADMIN"} {
		if _, err := db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT ?,?,password,?,0,10 FROM users WHERE id=1`, id+2, role, role); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 4,'internal',password,'INTERNAL',0,0 FROM users WHERE id=1;
		INSERT INTO user_task_submissions(id,user_id,task_id,status,submission_data,ai_review_count)
		VALUES(1,1,1,'PENDING','guest work',0),(2,4,1,'PENDING','unassigned internal work',0)`); err != nil {
		t.Fatal(err)
	}
	app := New(db)
	teacher, err := app.loginToken(2, "TEACHER", "TEACHER")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := app.loginToken(3, "SUPERADMIN", "SUPERADMIN")
	if err != nil {
		t.Fatal(err)
	}
	staleAdmin, err := app.loginToken(3, "SUPERADMIN", "TEACHER")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, body, token string
		status                    int
	}{
		{"GET", "/api/admin/submissions", "", teacher, 200},
		{"GET", "/api/submissions/2", "", teacher, 403},
		{"PUT", "/api/submissions/2/review", `{"status":"APPROVED"}`, teacher, 403},
		{"GET", "/api/submissions/2", "", admin, 200},
		{"PUT", "/api/submissions/2/review", `{"status":"APPROVED"}`, admin, 200},
		{"GET", "/api/admin/users", "", teacher, 403},
		{"GET", "/api/admin/users", "", staleAdmin, 401},
		{"GET", "/api/tutorials?levelId=1", "", teacher, 200},
		{"PUT", "/api/tutorials/1", `{"title":"Updated","levelId":2,"sortOrder":3}`, teacher, 403},
		{"DELETE", "/api/tutorials/1", "", teacher, 403},
		{"POST", "/api/admin/users/1/password-reset", "", teacher, 403},
		{"POST", "/api/admin/users/1/password-reset", "", admin, 400},
		{"DELETE", "/api/admin/users/3", "", admin, 400},
		{"PUT", "/api/admin/users/3/role?role=TEACHER", "", admin, 404},
		{"PUT", "/api/tutorials/1", `{"title":"Updated","levelId":2,"sortOrder":3}`, admin, 200},
		{"DELETE", "/api/tutorials/1", "", admin, 200},
		{"PUT", "/api/admin/users/1", `{"username":"alice","name":"Student","email":"student@example.test"}`, admin, 200},
	} {
		w := call(h, tc.method, tc.path, tc.body, tc.token)
		if w.Code != tc.status {
			t.Errorf("%s %s got %d, want %d: %s", tc.method, tc.path, w.Code, tc.status, w.Body)
		}
	}
	w := call(h, "GET", "/api/admin/submissions", "", teacher)
	if !bytes.Contains(w.Body.Bytes(), []byte("guest work")) || bytes.Contains(w.Body.Bytes(), []byte("unassigned internal work")) {
		t.Fatalf("teacher saw the wrong submissions: %s", w.Body)
	}
	w = call(h, "GET", "/api/admin/users?q=student%40example.test", "", admin)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("filtered users %d %s", w.Code, w.Body)
	}
}

func TestReviewAwardsXPOnce(t *testing.T) {
	db, h := testApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("mentor-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(2,'mentor',?,'TEACHER',0,10)`, string(hash)); err != nil {
		t.Fatal(err)
	}
	token := loginToken(t, h)
	w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"work"}`, token)
	if w.Code != 200 {
		t.Fatalf("submit %s", w.Body)
	}
	w = call(h, "POST", "/api/auth/login", `{"username":"mentor","password":"mentor-password"}`, "")
	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Data.Token == "" {
		t.Fatalf("mentor login %s", w.Body)
	}
	for i := 0; i < 2; i++ {
		w = call(h, "PUT", "/api/submissions/1/review", `{"status":"APPROVED","reviewComment":"ok"}`, result.Data.Token)
		if w.Code != 200 {
			t.Fatalf("review %d %s", w.Code, w.Body)
		}
	}
	w = call(h, "PUT", "/api/submissions/1/review", `{"status":"REJECTED","reviewComment":"changed"}`, result.Data.Token)
	if w.Code != 400 {
		t.Fatalf("review reversal accepted: %d", w.Code)
	}
	var xp, done int
	if err = db.QueryRow(`SELECT total_xp FROM users WHERE id=1`).Scan(&xp); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM user_level_progress WHERE user_id=1 AND level_id=1 AND status='COMPLETED'`).Scan(&done); err != nil {
		t.Fatal(err)
	}
	if xp != 15 || done != 1 {
		t.Fatalf("xp=%d completed=%d", xp, done)
	}
}

func TestRegistrationAndAdminAuthorization(t *testing.T) {
	_, h := testApp(t)
	w := call(h, "POST", "/api/auth/register", `{"username":"attacker","password":"password","role":"TEACHER"}`, "")
	if w.Code != 400 {
		t.Fatalf("self registration as teacher: %d %s", w.Code, w.Body)
	}
	w = call(h, "POST", "/api/auth/register", `{"username":"bob","password":"password","role":"EXTERNAL"}`, "")
	if w.Code != 200 {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	var response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	w = call(h, "GET", "/api/admin/users", "", response.Data.Token)
	if w.Code != 403 {
		t.Fatalf("non teacher admin: %d %s", w.Code, w.Body)
	}
}

func TestMentorshipAndChatPermissions(t *testing.T) {
	db, h := testApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("mentor-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(2,'mentor',?,'TEACHER',0,10),(3,'other',?,'EXTERNAL',0,0)`, string(hash), string(hash))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`UPDATE users SET role='INTERNAL' WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	student := loginToken(t, h)
	w := call(h, "POST", "/api/mentorship/request", `{"mentorId":2}`, student)
	if w.Code != 200 {
		t.Fatalf("request %d %s", w.Code, w.Body)
	}
	w = call(h, "POST", "/api/auth/login", `{"username":"mentor","password":"mentor-password"}`, "")
	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("mentor login")
	}
	mentor := result.Data.Token
	w = call(h, "PUT", "/api/mentorship/1/approve", "", mentor)
	if w.Code != 200 {
		t.Fatalf("approve %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", "/api/chat/conversations", "", student)
	if w.Code != 200 {
		t.Fatalf("conversations %d %s", w.Code, w.Body)
	}
	var list struct {
		Data []struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &list) != nil || len(list.Data) == 0 {
		t.Fatalf("conversation data %s", w.Body)
	}
	id := list.Data[0].ID
	w = call(h, "POST", "/api/chat/messages", fmt.Sprintf(`{"conversationId":%d,"content":"hello"}`, id), student)
	if w.Code != 200 {
		t.Fatalf("send %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", fmt.Sprintf("/api/chat/conversations/%d/messages", id), "", mentor)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("hello")) {
		t.Fatalf("history %d %s", w.Code, w.Body)
	}
	w = call(h, "POST", "/api/auth/login", `{"username":"other","password":"mentor-password"}`, "")
	if json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("other login")
	}
	w = call(h, "GET", fmt.Sprintf("/api/chat/conversations/%d/messages", id), "", result.Data.Token)
	if w.Code != 403 {
		t.Fatalf("outsider %d %s", w.Code, w.Body)
	}
	w = call(h, "PUT", "/api/mentorship/1/quit", "", student)
	if w.Code != 200 {
		t.Fatalf("quit %d %s", w.Code, w.Body)
	}
	w = call(h, "GET", fmt.Sprintf("/api/chat/conversations/%d/messages", id), "", student)
	if w.Code != 403 {
		t.Fatalf("ended member %d %s", w.Code, w.Body)
	}
}

func TestRejectCyclicPrerequisites(t *testing.T) {
	db, _ := testApp(t)
	one := int64(1)
	if err := validatePrerequisites(db, 1, &one, []int64{2}); err == nil {
		t.Fatal("accepted prerequisite cycle")
	}
	if err := validatePrerequisites(db, 1, &one, []int64{1}); err == nil {
		t.Fatal("accepted self prerequisite")
	}
	if err := validatePrerequisites(db, 1, &one, []int64{999}); err == nil {
		t.Fatal("accepted nonexistent prerequisite")
	}
}

func TestThirtyConcurrentReviewsGrantOnce(t *testing.T) {
	db, h := testApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("mentor-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(2,'mentor',?,'TEACHER',0,10)`, string(hash)); err != nil {
		t.Fatal(err)
	}
	student := loginToken(t, h)
	w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"work"}`, student)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = call(h, "POST", "/api/auth/login", `{"username":"mentor","password":"mentor-password"}`, "")
	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Data.Token == "" {
		t.Fatal("mentor login")
	}
	results := make(chan int, 30)
	for i := 0; i < 30; i++ {
		go func() {
			results <- call(h, "PUT", "/api/submissions/1/review", `{"status":"APPROVED","reviewComment":"ok"}`, result.Data.Token).Code
		}()
	}
	for i := 0; i < 30; i++ {
		if code := <-results; code != 200 {
			t.Fatalf("review returned %d", code)
		}
	}
	var xp, done, grants int
	if err = db.QueryRow(`SELECT total_xp FROM users WHERE id=1`).Scan(&xp); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM user_level_progress WHERE user_id=1 AND status='COMPLETED'`).Scan(&done); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM user_task_submissions WHERE user_id=1 AND task_id=1 AND xp_granted=1`).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if xp != 15 || done != 1 || grants != 1 {
		t.Fatalf("xp=%d done=%d grants=%d", xp, done, grants)
	}
}
