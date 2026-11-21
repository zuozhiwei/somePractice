package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"skilltree-go/internal/store"
)

func TestRegistrationSessionGenerationsAndInvitations(t *testing.T) {
	db, h := testApp(t)
	recoverySQL(t, db, `UPDATE users SET role='SUPERADMIN',level=1 WHERE id=1;
		INSERT INTO teams(id,name) VALUES(1,'Course team');
		INSERT INTO invite_codes(code,team_id,active,expires_at) VALUES('TEAM',1,1,'2099-01-01')`)
	admin := loginToken(t, h)
	var invitation string
	learningResponse(t, h, "GET", "/api/mentorship/invite-link", "", admin, &invitation)
	register := func(name, role, key, credential string) (string, int64) {
		t.Helper()
		v := map[string]string{"username": name, "role": role, "password": "test-password"}
		if key != "" {
			v[key] = credential
		}
		payload, _ := json.Marshal(v)
		var result struct {
			Token string
			User  struct{ ID int64 }
		}
		learningResponse(t, h, "POST", "/api/auth/register", string(payload), "", &result)
		learningResponse(t, h, "GET", "/api/auth/profile", "", result.Token, nil)
		if testSessionVersion(t, db, result.User.ID) <= 0 {
			t.Fatal("new account needs a positive version")
		}
		return result.Token, result.User.ID
	}
	register("invited", "INTERNAL", "inviteToken", invitation)
	register("team-student", "INTERNAL", "teamInviteCode", "TEAM")
	old, id := register("learner", "EXTERNAL", "", "")

	server := httptest.NewServer(h)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws?token="+old, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	learningResponse(t, h, "DELETE", "/api/admin/users/"+strconv.FormatInt(id, 10), "", admin, nil)
	if _, _, err := c.Read(ctx); err == nil || ctx.Err() != nil {
		t.Fatal("deletion left the old connection open")
	}
	fresh, sameID := register("learner", "INTERNAL", "teamInviteCode", "TEAM")
	if sameID != id || fresh == old {
		t.Fatal("expected a new session under a reused rowid")
	}
	if w := call(h, "GET", "/api/auth/profile", "", old); w.Code != 401 {
		t.Fatalf("previous session status: %d", w.Code)
	}

	var oldInvite string
	learningResponse(t, h, "PUT", "/api/admin/users/"+strconv.FormatInt(sameID, 10)+"/level?level=1", "", admin, nil)
	learningResponse(t, h, "GET", "/api/mentorship/invite-link", "", fresh, &oldInvite)
	learningResponse(t, h, "DELETE", "/api/admin/users/"+strconv.FormatInt(sameID, 10), "", admin, nil)
	register("learner", "INTERNAL", "teamInviteCode", "TEAM")
	body, _ := json.Marshal(map[string]string{"username": "later", "password": "test-password", "role": "INTERNAL", "inviteToken": oldInvite})
	if w := call(h, "POST", "/api/auth/register", string(body), ""); w.Code != 400 {
		t.Fatalf("invitation from deleted mentor: %d", w.Code)
	}
}

func TestExistingUnversionedAccountsSurviveUpgrade(t *testing.T) {
	db, h := testApp(t)
	// An existing installation has user rows before session generation is introduced.
	recoverySQL(t, db, "DELETE FROM login_versions WHERE user_id=1")
	old, err := New(db).loginToken(1, "alice", "EXTERNAL", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	learningResponse(t, h, "GET", "/api/auth/profile", "", old, nil)
	loginToken(t, h)
	recoverySQL(t, db, "DELETE FROM users WHERE id=1")
	recoverySQL(t, db, "INSERT INTO users(id,username,password,role,total_xp) VALUES(1,'alice','test-only','EXTERNAL',0)")
	if testSessionVersion(t, db, 1) <= 0 {
		t.Fatal("directly created users need a new session generation")
	}
	if w := call(h, "GET", "/api/auth/profile", "", old); w.Code != 401 {
		t.Fatalf("unversioned session survived account replacement: %d", w.Code)
	}
}
