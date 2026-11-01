package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"golang.org/x/crypto/bcrypt"
)

func TestChatPushAndCursorCatchup(t *testing.T) {
	db, h := testApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("mentor-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(2,'mentor',?,'TEACHER',0,10)`, string(hash)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO mentorships(mentor_id,apprentice_id,status) VALUES(2,1,'ACTIVE')`); err != nil {
		t.Fatal(err)
	}
	student := loginToken(t, h)
	w := call(h, "GET", "/api/chat/conversations", "", student)
	var list struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Data) == 0 {
		t.Fatalf("conversation: %s", w.Body)
	}
	cid := list.Data[0].ID
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?token="+student, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	w = call(h, "POST", "/api/chat/messages", fmt.Sprintf(`{"conversationId":%d,"content":"one"}`, cid), student)
	if w.Code != 200 {
		t.Fatalf("send %s", w.Body)
	}
	var evt struct {
		Type                      string `json:"type"`
		ConversationID, MessageID int64
	}
	if err = wsjson.Read(ctx, conn, &evt); err != nil {
		t.Fatal(err)
	}
	if evt.Type != "changed" || evt.ConversationID != cid || evt.MessageID <= 0 {
		t.Fatalf("event: %+v", evt)
	}
	w = call(h, "GET", fmt.Sprintf("/api/chat/conversations/%d/messages?afterId=0&size=100", cid), "", student)
	var got struct {
		Data struct {
			Content []struct {
				ID int64 `json:"id"`
			} `json:"content"`
		} `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &got) != nil || len(got.Data.Content) != 1 || got.Data.Content[0].ID != evt.MessageID {
		t.Fatalf("cursor result %s", w.Body)
	}
}

func TestThirtyWebSocketRecipients(t *testing.T) {
	db, h := testApp(t)
	if _, err := db.Exec(`UPDATE users SET role='TEACHER',level=50 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversations(id,type,owner_id) VALUES(1,'GROUP',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(1,1)`); err != nil {
		t.Fatal(err)
	}
	for id := 2; id <= 31; id++ {
		if _, err := db.Exec(`INSERT INTO users(id,username,password,role,total_xp,level) VALUES(?,?,?,'INTERNAL',0,0)`, id, fmt.Sprintf("student%d", id), "hash"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO mentorships(mentor_id,apprentice_id,status) VALUES(1,?,'ACTIVE')`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(1,?)`, id); err != nil {
			t.Fatal(err)
		}
	}
	app := New(db)
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conns := make([]*websocket.Conn, 0, 30)
	for id := 2; id <= 31; id++ {
		token, err := app.loginToken(int64(id), fmt.Sprintf("student%d", id), "INTERNAL")
		if err != nil {
			t.Fatal(err)
		}
		conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?token="+token, nil)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, c := range conns {
			c.CloseNow()
		}
	}()
	teacher := loginToken(t, h)
	w := call(h, "POST", "/api/chat/messages", `{"conversationId":1,"content":"announcement"}`, teacher)
	if w.Code != 200 {
		t.Fatalf("send %s", w.Body)
	}
	results := make(chan error, len(conns))
	for _, conn := range conns {
		go func(c *websocket.Conn) {
			var e struct {
				Type           string `json:"type"`
				ConversationID int64  `json:"conversationId"`
			}
			err := wsjson.Read(ctx, c, &e)
			if err == nil && (e.Type != "changed" || e.ConversationID != 1) {
				err = fmt.Errorf("invalid event")
			}
			results <- err
		}(conn)
	}
	for range conns {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}
