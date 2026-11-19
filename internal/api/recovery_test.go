package api

import (
	"context"
	"net/http/httptest"
	"time"

	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"strings"
	"testing"
)

func recoveryHandler(app *App) http.Handler {
	m := http.NewServeMux()
	app.Register(m)
	return m
}
func recoverySQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func resetWith(h http.Handler, identifier, method, code string) int {
	b, _ := json.Marshal(map[string]string{"identifier": identifier, "method": method, "code": code, "newPassword": "after-recovery"})
	return call(h, "POST", "/api/auth/reset-password", string(b), "").Code
}

func TestEmailRecoveryRestartAndSessionRevocation(t *testing.T) {
	db, original := testApp(t)
	oldToken := loginToken(t, original)
	recoverySQL(t, db, "UPDATE users SET email=' ALICE@Example.test ' WHERE id=1")
	app := New(db)
	code := ""
	app.sendRecoveryMail = func(to, value string) error {
		if to != "ALICE@Example.test" {
			t.Errorf("recipient mismatch")
		}
		code = value
		return nil
	}
	h := recoveryHandler(app)
	if w := call(h, "POST", "/api/auth/forgot-password", `{"identifier":" alice "}`, ""); w.Code != 200 || len(code) != 6 {
		t.Fatalf("email issue status %d", w.Code)
	}
	var digest []byte
	if err := db.QueryRow("SELECT digest FROM password_recovery WHERE user_id=1").Scan(&digest); err != nil || len(digest) != 32 {
		t.Fatal("missing keyed digest", err)
	}
	h = recoveryHandler(New(db))
	server := httptest.NewServer(h)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws?token="+oldToken, nil)
	if err != nil {
		t.Fatal("opening signed-in socket", err)
	}
	defer socket.CloseNow()
	results := make(chan int, 2)
	for range 2 {
		go func() { results <- resetWith(h, " alice@example.TEST ", "email", code) }()
	}
	first, second := <-results, <-results
	if first+second != 600 || (first != 200 && second != 200) {
		t.Fatalf("concurrent use: %d %d", first, second)
	}
	if _, _, err := socket.Read(ctx); err == nil || ctx.Err() != nil {
		t.Fatal("reset did not close the previous WebSocket session")
	}
	if w := call(h, "GET", "/api/auth/profile", "", oldToken); w.Code != 401 {
		t.Fatalf("old session status %d", w.Code)
	}
	if w := call(h, "POST", "/api/auth/login", `{"username":"alice","password":"after-recovery"}`, ""); w.Code != 200 {
		t.Fatalf("new password status %d", w.Code)
	}
	if got := resetWith(h, "alice", "email", code); got != 400 {
		t.Fatalf("replay status %d", got)
	}
}

func TestEmailRecoveryDeliveryAndIdentityChanges(t *testing.T) {
	db, _ := testApp(t)
	app := New(db)
	h := recoveryHandler(app)
	payload := `{"email":" Student@EXAMPLE.test "}`
	recoverySQL(t, db, "UPDATE users SET email='student@example.test' WHERE id=1")
	app.sendRecoveryMail = func(to, code string) error { return errors.New("delivery failed") }
	if w := call(h, "POST", "/api/auth/forgot-password", payload, ""); w.Code != 502 {
		t.Fatalf("delivery failure %d", w.Code)
	}
	var code, first string
	count := 0
	app.sendRecoveryMail = func(to, value string) error { code = value; count++; return nil }
	for range 2 {
		if w := call(h, "POST", "/api/auth/forgot-password", payload, ""); w.Code != 200 {
			t.Fatalf("resend %d", w.Code)
		}
	}
	if count != 1 {
		t.Fatalf("cooldown did not limit sends: %d", count)
	}
	first = code
	recoverySQL(t, db, "UPDATE password_recovery SET issued_at=issued_at-61")
	call(h, "POST", "/api/auth/forgot-password", payload, "")
	if count != 2 {
		t.Fatal("missing second delivery")
	}
	if code != first && resetWith(h, "alice", "email", first) != 400 {
		t.Fatal("previous code accepted")
	}
	recoverySQL(t, db, "UPDATE users SET email=NULL WHERE id=1")
	recoverySQL(t, db, `INSERT INTO users(id,username,password,email,role,total_xp,level) SELECT 2,'recipient',password,'student@example.test','EXTERNAL',0,0 FROM users WHERE id=1`)
	if got := resetWith(h, "student@example.test", "email", code); got != 400 {
		t.Fatalf("reassigned email status %d", got)
	}
	// Reproduce a database imported before equivalent email spellings were checked.
	recoverySQL(t, db, "DROP TRIGGER recovery_unique_email_update")
	recoverySQL(t, db, "UPDATE users SET email=' STUDENT@EXAMPLE.TEST ' WHERE id=1")
	for _, who := range []string{"student@example.test", "alice", "no-such-user"} {
		w := call(h, "POST", "/api/auth/forgot-password", fmt.Sprintf(`{"identifier":%q}`, who), "")
		if w.Code != 200 || count != 2 || !strings.Contains(w.Body.String(), emailRequestMessage) {
			t.Fatalf("ambiguous or unknown identity: %d", w.Code)
		}
	}
}

func TestAdminAssistedRecovery(t *testing.T) {
	db, h := testApp(t)
	recoverySQL(t, db, `INSERT INTO users(id,username,password,role,total_xp,level) SELECT 2,'ops',password,'SUPERADMIN',0,0 FROM users WHERE id=1`)
	admin, err := New(db).loginToken(2, "ops", "SUPERADMIN", testSessionVersion(t, db, 2))
	if err != nil {
		t.Fatal(err)
	}
	user := loginToken(t, h)
	issue := func(t *testing.T) string {
		t.Helper()
		w := call(h, "POST", "/api/admin/users/1/recovery-code", `{"identityVerified":true}`, admin)
		var r struct{ Data struct{ RecoveryCode string } }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &r) != nil || len(r.Data.RecoveryCode) != 19 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("admin issuance status %d", w.Code)
		}
		return r.Data.RecoveryCode
	}
	if w := call(h, "POST", "/api/admin/users/1/recovery-code", `{"identityVerified":true}`, user); w.Code != 403 {
		t.Fatal("non-admin was allowed")
	}
	if w := call(h, "POST", "/api/admin/users/2/recovery-code", `{"identityVerified":true}`, admin); w.Code != 403 {
		t.Fatal("protected account was eligible")
	}
	if w := call(h, "POST", "/api/admin/users/1/recovery-code", "{}", admin); w.Code != 400 {
		t.Fatal("identity confirmation was skipped")
	}
	code := issue(t)
	if w := call(h, "GET", "/api/auth/profile", "", user); w.Code != 200 {
		t.Fatal("issuance interrupted the student")
	}
	latest := issue(t)
	if got := resetWith(h, "alice", "admin", code); got != 400 {
		t.Fatal("superseded code accepted")
	}
	recoverySQL(t, db, "UPDATE password_recovery SET expires_at=0")
	if got := resetWith(h, "alice", "admin", latest); got != 400 {
		t.Fatal("expired code accepted")
	}
	latest = issue(t)
	for range 5 {
		resetWith(h, "alice", "admin", "INVALID")
	}
	if got := resetWith(recoveryHandler(New(db)), "alice", "admin", latest); got != 400 {
		t.Fatal("failed attempts reset on restart")
	}
	latest = issue(t)
	if got := resetWith(h, "alice", "admin", strings.ToLower(latest)); got != 200 {
		t.Fatalf("offline recovery %d", got)
	}
	if got := resetWith(h, "alice", "admin", latest); got != 400 {
		t.Fatal("offline code reused")
	}
}
