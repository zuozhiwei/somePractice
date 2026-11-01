package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIReviewWithMockService(t *testing.T) {
	db, h := testApp(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fake-key" {
			t.Error("missing AI token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"pass\":true,\"feedback\":\"good\",\"score\":90}"}}]}`))
	}))
	defer fake.Close()
	t.Setenv("DEEPSEEK_API_KEY", "fake-key")
	t.Setenv("DEEPSEEK_API_URL", fake.URL)
	if _, err := db.Exec(`UPDATE tasks SET ai_review_enabled=1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	token := loginToken(t, h)
	w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"my work"}`, token)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = call(h, "POST", "/api/submissions/1/ai-review", "", token)
	if w.Code != 200 {
		t.Fatalf("review %d %s", w.Code, w.Body)
	}
	var xp, done int
	if err := db.QueryRow(`SELECT total_xp FROM users WHERE id=1`).Scan(&xp); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM user_level_progress WHERE user_id=1 AND level_id=1 AND status='COMPLETED'`).Scan(&done); err != nil {
		t.Fatal(err)
	}
	if xp != 15 || done != 1 {
		t.Fatalf("xp %d done %d", xp, done)
	}
}
func TestQiniuUploadPolicyAndPrivateURL(t *testing.T) {
	_, h := testApp(t)
	t.Setenv("QINIU_ACCESS_KEY", "test-access")
	t.Setenv("QINIU_SECRET_KEY", "test-secret")
	t.Setenv("QINIU_BUCKET", "test-bucket")
	t.Setenv("QINIU_DOMAIN", "https://files.example.test")
	token := loginToken(t, h)
	w := call(h, "GET", "/api/upload/token", "", token)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	var v struct {
		Data struct {
			Token     string `json:"token"`
			KeyPrefix string `json:"keyPrefix"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &v) != nil {
		t.Fatal("invalid response")
	}
	parts := strings.Split(v.Data.Token, ":")
	if len(parts) != 3 || parts[1] != qiniuSign(parts[2]) || v.Data.KeyPrefix != "uploads/1/" {
		t.Fatal("upload signature invalid")
	}
	raw, err := base64.URLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if json.Unmarshal(raw, &policy) != nil || policy["scope"] != "test-bucket:uploads/1/" || policy["fsizeLimit"] != float64(10485760) {
		t.Fatal("policy incorrect")
	}
	w = call(h, "POST", "/api/files/sign-urls", `{"urls":["https://files.example.test/uploads/1/a.png","https://files.example.test/uploads/2/private.png"]}`, token)
	var urls struct {
		Data map[string]string `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &urls) != nil {
		t.Fatal(w.Body)
	}
	if !strings.Contains(urls.Data["https://files.example.test/uploads/1/a.png"], "token=") || urls.Data["https://files.example.test/uploads/2/private.png"] != "https://files.example.test/uploads/2/private.png" {
		t.Fatal("file signing access mismatch")
	}
}

func TestAIReviewLimitAcrossSubmissionsAndFailureRefund(t *testing.T) {
	db, h := testApp(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", http.StatusBadGateway) }))
	defer fake.Close()
	t.Setenv("DEEPSEEK_API_KEY", "fake")
	t.Setenv("DEEPSEEK_API_URL", fake.URL)
	if _, err := db.Exec(`UPDATE tasks SET ai_review_enabled=1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	token := loginToken(t, h)
	if w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"one"}`, token); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w := call(h, "POST", "/api/submissions/1/ai-review", "", token); w.Code != 502 {
		t.Fatalf("failed AI %d %s", w.Code, w.Body)
	}
	var n int
	if err := db.QueryRow(`SELECT ai_review_count FROM user_task_submissions WHERE id=1`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("attempt not refunded: %d %v", n, err)
	}
	if _, err := db.Exec(`UPDATE user_task_submissions SET ai_review_count=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if w := call(h, "POST", "/api/tasks/1/submit", `{"submissionData":"two"}`, token); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w := call(h, "POST", "/api/submissions/2/ai-review", "", token); w.Code != 400 {
		t.Fatalf("task limit %d %s", w.Code, w.Body)
	}
}
