package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const chatAccess = `EXISTS(SELECT 1 FROM conversation_members cm JOIN conversations c ON c.id=cm.conversation_id WHERE cm.conversation_id=? AND cm.user_id=? AND ((c.type='GROUP' AND (c.owner_id=cm.user_id OR EXISTS(SELECT 1 FROM mentorships m WHERE m.mentor_id=c.owner_id AND m.apprentice_id=cm.user_id AND m.status='ACTIVE'))) OR (c.type='DIRECT' AND EXISTS(SELECT 1 FROM conversation_members other JOIN mentorships m ON m.status='ACTIVE' AND ((m.mentor_id=cm.user_id AND m.apprentice_id=other.user_id) OR (m.apprentice_id=cm.user_id AND m.mentor_id=other.user_id)) WHERE other.conversation_id=c.id))))`

func (a *App) chatAllowed(r *http.Request, conversation, user int64) (bool, error) {
	var allowed bool
	err := a.DB.QueryRowContext(r.Context(), `SELECT `+chatAccess, conversation, user).Scan(&allowed)
	return allowed, err
}
func (a *App) chatEnsure(r *http.Request, user int64) error {
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(r.Context(), `SELECT mentor_id,apprentice_id FROM mentorships WHERE status='ACTIVE' AND (mentor_id=? OR apprentice_id=?) AND mentor_id IS NOT NULL AND apprentice_id IS NOT NULL`, user, user)
	if err != nil {
		return err
	}
	pairs := [][2]int64{}
	for rows.Next() {
		var x, y int64
		if err = rows.Scan(&x, &y); err != nil {
			rows.Close()
			return err
		}
		pairs = append(pairs, [2]int64{x, y})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	for _, pair := range pairs {
		mentor, student := pair[0], pair[1]
		var cid int64
		err = tx.QueryRow(`SELECT id FROM conversations WHERE type='GROUP' AND owner_id=? LIMIT 1`, mentor).Scan(&cid)
		if err == sql.ErrNoRows {
			res, e := tx.Exec(`INSERT INTO conversations(type,owner_id,created_at) VALUES('GROUP',?,?)`, mentor, now)
			if e != nil {
				return e
			}
			cid, err = res.LastInsertId()
		}
		if err != nil {
			return err
		}
		for _, uid := range pair {
			_, err = tx.Exec(`INSERT INTO conversation_members(conversation_id,user_id,joined_at,last_read_at) SELECT ?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=? AND user_id=?)`, cid, uid, now, now, cid, uid)
			if err != nil {
				return err
			}
		}
		err = tx.QueryRow(`SELECT c.id FROM conversations c JOIN conversation_members m ON m.conversation_id=c.id AND m.user_id=? JOIN conversation_members n ON n.conversation_id=c.id AND n.user_id=? WHERE c.type='DIRECT' LIMIT 1`, mentor, student).Scan(&cid)
		if err == sql.ErrNoRows {
			res, e := tx.Exec(`INSERT INTO conversations(type,created_at) VALUES('DIRECT',?)`, now)
			if e != nil {
				return e
			}
			cid, err = res.LastInsertId()
		}
		if err != nil {
			return err
		}
		for _, uid := range pair {
			_, err = tx.Exec(`INSERT INTO conversation_members(conversation_id,user_id,joined_at,last_read_at) SELECT ?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=? AND user_id=?)`, cid, uid, now, now, cid, uid)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (a *App) chatConversations(w http.ResponseWriter, r *http.Request, user int64) {
	if err := a.chatEnsure(r, user); err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT c.id,c.type,c.owner_id,cm.last_read_at FROM conversations c JOIN conversation_members cm ON cm.conversation_id=c.id WHERE cm.user_id=? ORDER BY c.id`, user)
	if err != nil {
		fail(w)
		return
	}
	type conv struct {
		id    int64
		typ   string
		owner sql.NullInt64
		read  sql.NullString
	}
	var cs []conv
	for rows.Next() {
		var c conv
		if rows.Scan(&c.id, &c.typ, &c.owner, &c.read) != nil {
			rows.Close()
			fail(w)
			return
		}
		cs = append(cs, c)
	}
	if rows.Err() != nil {
		rows.Close()
		fail(w)
		return
	}
	rows.Close()
	out := []any{}
	for _, c := range cs {
		allowed, err := a.chatAllowed(r, c.id, user)
		if err != nil {
			fail(w)
			return
		}
		if !allowed {
			continue
		}
		name := "会话"
		var other any
		if c.typ == "GROUP" {
			name = "师门群聊"
			var mentor sql.NullString
			if c.owner.Valid && a.DB.QueryRowContext(r.Context(), `SELECT COALESCE(NULLIF(name,''),username) FROM users WHERE id=?`, c.owner.Int64).Scan(&mentor) == nil && mentor.Valid {
				name += "（" + mentor.String + "）"
			}
		} else {
			var oid int64
			var nm string
			if a.DB.QueryRowContext(r.Context(), `SELECT u.id,COALESCE(NULLIF(u.name,''),u.username) FROM conversation_members m JOIN users u ON u.id=m.user_id WHERE m.conversation_id=? AND m.user_id<>? LIMIT 1`, c.id, user).Scan(&oid, &nm) == nil {
				name = nm
				other = oid
			}
		}
		var unread int
		if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM chat_messages WHERE conversation_id=? AND sender_id<>? AND created_at>COALESCE(?, '')`, c.id, user, nullString(c.read)).Scan(&unread); err != nil {
			fail(w)
			return
		}
		var content, created, sender sql.NullString
		err = a.DB.QueryRowContext(r.Context(), `SELECT m.content,m.created_at,COALESCE(NULLIF(u.name,''),u.username,'') FROM chat_messages m LEFT JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=? ORDER BY m.created_at DESC,m.id DESC LIMIT 1`, c.id).Scan(&content, &created, &sender)
		if err != nil && err != sql.ErrNoRows {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": c.id, "type": c.typ, "name": name, "otherUserId": other, "unreadCount": unread, "lastMessage": nullString(content), "lastMessageAt": a.timestamp(created), "lastSenderName": nullString(sender)})
	}
	ok(w, out)
}
func (a *App) chatMessages(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "会话无效")
		return
	}
	yes, err := a.chatAllowed(r, id, user)
	if err != nil {
		fail(w)
		return
	}
	if !yes {
		reply(w, 403, nil, "无权查看该会话")
		return
	}
	page, size := 0, 50
	if v := r.URL.Query().Get("page"); v != "" {
		page, err = strconv.Atoi(v)
		if err != nil || page < 0 || page > 1000000 {
			bad(w, "页码无效")
			return
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		size, err = strconv.Atoi(v)
		if err != nil || size < 1 || size > 100 {
			bad(w, "数量无效")
			return
		}
	}
	if raw := r.URL.Query().Get("afterId"); raw != "" {
		after, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || after < 0 {
			bad(w, "消息游标无效")
			return
		}
		rows, e := a.DB.QueryContext(r.Context(), `SELECT m.id,m.sender_id,COALESCE(NULLIF(u.name,''),u.username,'用户'),COALESCE(u.role,''),m.content,m.created_at FROM chat_messages m LEFT JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=? AND m.id>? ORDER BY m.id ASC LIMIT ?`, id, after, size)
		if e != nil {
			fail(w)
			return
		}
		defer rows.Close()
		out := []any{}
		for rows.Next() {
			var mid, sender int64
			var name, role, content string
			var created sql.NullString
			if rows.Scan(&mid, &sender, &name, &role, &content, &created) != nil {
				fail(w)
				return
			}
			out = append(out, map[string]any{"id": mid, "conversationId": id, "senderId": sender, "senderName": name, "senderRole": role, "content": content, "createdAt": a.timestamp(created)})
		}
		if rows.Err() != nil {
			fail(w)
			return
		}
		ok(w, map[string]any{"content": out})
		return
	}
	var total int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM chat_messages WHERE conversation_id=?`, id).Scan(&total); err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT m.id,m.sender_id,COALESCE(NULLIF(u.name,''),u.username,'用户'),COALESCE(u.role,''),m.content,m.created_at FROM chat_messages m LEFT JOIN users u ON u.id=m.sender_id WHERE m.conversation_id=? ORDER BY m.created_at DESC,m.id DESC LIMIT ? OFFSET ?`, id, size, page*size)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var mid, sender int64
		var name, role, content string
		var created sql.NullString
		if rows.Scan(&mid, &sender, &name, &role, &content, &created) != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": mid, "conversationId": id, "senderId": sender, "senderName": name, "senderRole": role, "content": content, "createdAt": a.timestamp(created)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"content": out, "total": total})
}
func (a *App) chatRead(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "会话无效")
		return
	}
	yes, err := a.chatAllowed(r, id, user)
	if err != nil {
		fail(w)
		return
	}
	if !yes {
		reply(w, 403, nil, "无权操作")
		return
	}
	_, err = a.DB.ExecContext(r.Context(), `UPDATE conversation_members SET last_read_at=? WHERE conversation_id=? AND user_id=?`, time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999"), id, user)
	if err != nil {
		fail(w)
		return
	}
	ok(w, nil)
}
func (a *App) chatUnread(w http.ResponseWriter, r *http.Request, user int64) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT conversation_id,COALESCE(last_read_at,'') FROM conversation_members WHERE user_id=?`, user)
	if err != nil {
		fail(w)
		return
	}
	type item struct {
		id   int64
		last string
	}
	var list []item
	for rows.Next() {
		var v item
		if rows.Scan(&v.id, &v.last) != nil {
			rows.Close()
			fail(w)
			return
		}
		list = append(list, v)
	}
	if rows.Err() != nil {
		rows.Close()
		fail(w)
		return
	}
	rows.Close()
	total := 0
	for _, v := range list {
		yes, err := a.chatAllowed(r, v.id, user)
		if err != nil {
			fail(w)
			return
		}
		if !yes {
			continue
		}
		var n int
		if a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM chat_messages WHERE conversation_id=? AND sender_id<>? AND created_at>?`, v.id, user, v.last).Scan(&n) != nil {
			fail(w)
			return
		}
		total += n
	}
	ok(w, total)
}
func (a *App) chatSend(w http.ResponseWriter, r *http.Request, user int64) {
	var req struct {
		ConversationID int64  `json:"conversationId"`
		Content        string `json:"content"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	if json.NewDecoder(r.Body).Decode(&req) != nil || len([]rune(strings.TrimSpace(req.Content))) == 0 || len([]rune(req.Content)) > 4000 {
		bad(w, "消息内容无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var yes bool
	err = tx.QueryRowContext(r.Context(), `SELECT `+chatAccess, req.ConversationID, user).Scan(&yes)
	if err != nil {
		fail(w)
		return
	}
	if !yes {
		reply(w, 403, nil, "无权操作该会话")
		return
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	res, err := tx.ExecContext(r.Context(), `INSERT INTO chat_messages(conversation_id,sender_id,content,created_at) VALUES(?,?,?,?)`, req.ConversationID, user, strings.TrimSpace(req.Content), now)
	if err != nil {
		fail(w)
		return
	}
	id, _ := res.LastInsertId()
	var name, role string
	if err = tx.QueryRowContext(r.Context(), `SELECT COALESCE(NULLIF(name,''),username),COALESCE(role,'') FROM users WHERE id=?`, user).Scan(&name, &role); err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id, "conversationId": req.ConversationID, "senderId": user, "senderName": name, "senderRole": role, "content": strings.TrimSpace(req.Content), "createdAt": a.timestamp(sql.NullString{String: now, Valid: true})})
	a.notifyChat(chatEvent{ConversationID: req.ConversationID, MessageID: id})
}

type chatEvent struct {
	ConversationID int64
	MessageID      int64
}

func (a *App) notifyChat(event chatEvent) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	for _, clients := range a.chatClients {
		for ch := range clients {
			select {
			case ch <- event:
			default:
			}
		}
	}
}
func (a *App) chatWS(w http.ResponseWriter, r *http.Request, user int64) {
	a.chatMu.Lock()
	if len(a.chatClients[user]) >= 3 {
		a.chatMu.Unlock()
		reply(w, 429, nil, "连接过多")
		return
	}
	ch := make(chan chatEvent, 16)
	if a.chatClients[user] == nil {
		a.chatClients[user] = map[chan chatEvent]struct{}{}
	}
	a.chatClients[user][ch] = struct{}{}
	a.chatMu.Unlock()
	defer func() {
		a.chatMu.Lock()
		delete(a.chatClients[user], ch)
		if len(a.chatClients[user]) == 0 {
			delete(a.chatClients, user)
		}
		a.chatMu.Unlock()
	}()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithTimeout(r.Context(), time.Hour)
	defer cancel()
	ctx = conn.CloseRead(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			yes, err := a.chatAllowed(r, event.ConversationID, user)
			if err != nil || !yes {
				continue
			}
			writeCtx, end := context.WithTimeout(ctx, 5*time.Second)
			err = wsjson.Write(writeCtx, conn, map[string]any{"type": "changed", "conversationId": event.ConversationID, "messageId": event.MessageID})
			end()
			if err != nil {
				return
			}
		}
	}
}
