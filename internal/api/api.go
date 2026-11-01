package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	DB          *sql.DB
	key         []byte
	zone        *time.Location
	resetMu     sync.Mutex
	resets      map[string]resetEntry
	chatMu      sync.Mutex
	contentMu   sync.Mutex
	rateMu      sync.Mutex
	rates       map[string]rateWindow
	bcryptSlots chan struct{}
	chatClients map[int64]map[chan chatEvent]struct{}
}
type claims struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func New(db *sql.DB) *App {
	b, _ := base64.StdEncoding.DecodeString(os.Getenv("SKILLTREE_JWT_SECRET"))
	zone := time.UTC
	if name := os.Getenv("SKILLTREE_TIMEZONE"); name != "" {
		if z, err := time.LoadLocation(name); err == nil {
			zone = z
		}
	}
	return &App{DB: db, key: b, zone: zone, resets: make(map[string]resetEntry), rates: make(map[string]rateWindow), bcryptSlots: make(chan struct{}, 4), chatClients: make(map[int64]map[chan chatEvent]struct{})}
}
func (a *App) timestamp(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05.999999999", v.String, a.zone)
	if err != nil {
		return nil
	}
	return t.UnixMilli()
}
func (a *App) Register(m *http.ServeMux) {
	m.HandleFunc("POST /api/auth/login", a.login)
	m.HandleFunc("POST /api/auth/register", a.register)
	m.HandleFunc("PUT /api/auth/profile", a.auth(a.updateProfile))
	m.HandleFunc("POST /api/auth/forgot-password", a.forgotPassword)
	m.HandleFunc("POST /api/auth/reset-password", a.resetPassword)
	m.HandleFunc("GET /api/upload/token", a.auth(a.uploadToken))
	m.HandleFunc("POST /api/files/sign-urls", a.auth(a.signURLs))
	m.HandleFunc("GET /api/auth/profile", a.auth(a.profile))
	m.HandleFunc("GET /api/maps", a.auth(a.maps))
	m.HandleFunc("GET /api/maps/progress", a.auth(a.mapProgress))
	m.HandleFunc("GET /api/maps/{id}", a.auth(a.mapDetail))
	m.HandleFunc("GET /api/levels/tree", a.auth(a.levelTree))
	m.HandleFunc("GET /api/levels/{id}", a.auth(a.levelDetail))
	m.HandleFunc("GET /api/tasks", a.auth(a.taskList))
	m.HandleFunc("POST /api/tasks/{id}/submit", a.auth(a.submit))
	m.HandleFunc("GET /api/tasks/{id}/submissions", a.auth(a.submissions))
	m.HandleFunc("GET /api/submissions/my", a.auth(a.mySubmissions))
	m.HandleFunc("GET /api/submissions/apprentice", a.auth(a.apprenticeSubmissions))
	m.HandleFunc("GET /api/submissions/external", a.auth(a.externalSubmissions))
	m.HandleFunc("GET /api/submissions/{id}", a.auth(a.submissionDetail))
	m.HandleFunc("POST /api/submissions/{id}/ai-review", a.auth(a.aiReview))
	m.HandleFunc("PUT /api/submissions/{id}/review", a.auth(a.review))
	m.HandleFunc("PUT /api/levels/{id}/complete", a.auth(a.complete))
	m.HandleFunc("GET /api/mentor/dialog", a.auth(a.mentorDialog))
	m.HandleFunc("GET /api/adventure/mission", a.auth(a.mission))
	m.HandleFunc("GET /api/leaderboard/global", a.auth(a.leaderboard))
	m.HandleFunc("GET /api/leaderboard/team", a.auth(a.teamLeaderboard))
	m.HandleFunc("GET /api/mentorship/invite-link", a.auth(a.inviteLink))
	m.HandleFunc("GET /api/mentorship/my", a.auth(a.mentorshipMy))
	m.HandleFunc("GET /api/mentorship/fellows", a.auth(a.mentorshipFellows))
	m.HandleFunc("GET /api/mentorship/apprentices", a.auth(a.mentorshipApprentices))
	m.HandleFunc("GET /api/mentorship/search-users", a.auth(a.mentorshipSearch))
	m.HandleFunc("POST /api/mentorship/request", a.auth(a.mentorshipRequest))
	m.HandleFunc("PUT /api/mentorship/{id}/approve", a.auth(a.mentorshipAction))
	m.HandleFunc("PUT /api/mentorship/{id}/reject", a.auth(a.mentorshipAction))
	m.HandleFunc("PUT /api/mentorship/{id}/quit", a.auth(a.mentorshipAction))
	m.HandleFunc("DELETE /api/mentorship/{id}/cancel", a.auth(a.mentorshipAction))
	m.HandleFunc("GET /api/chat/conversations", a.auth(a.chatConversations))
	m.HandleFunc("GET /api/chat/conversations/{id}/messages", a.auth(a.chatMessages))
	m.HandleFunc("PUT /api/chat/conversations/{id}/read", a.auth(a.chatRead))
	m.HandleFunc("GET /api/chat/unread", a.auth(a.chatUnread))
	m.HandleFunc("POST /api/chat/messages", a.auth(a.chatSend))
	m.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+r.URL.Query().Get("token"))
		a.auth(a.chatWS)(w, r)
	})
	m.HandleFunc("GET /api/admin/levels", a.auth(a.teacher(a.adminLevels)))
	m.HandleFunc("GET /api/admin/levels/{id}", a.auth(a.teacher(a.adminLevels)))
	m.HandleFunc("POST /api/admin/levels", a.auth(a.teacher(a.adminSaveLevel)))
	m.HandleFunc("PUT /api/admin/levels/{id}", a.auth(a.teacher(a.adminSaveLevel)))
	m.HandleFunc("DELETE /api/admin/levels/{id}", a.auth(a.teacher(a.adminDeleteContent)))
	m.HandleFunc("POST /api/admin/tasks", a.auth(a.teacher(a.adminSaveTask)))
	m.HandleFunc("PUT /api/admin/tasks/{id}", a.auth(a.teacher(a.adminSaveTask)))
	m.HandleFunc("DELETE /api/admin/tasks/{id}", a.auth(a.teacher(a.adminDeleteContent)))
	m.HandleFunc("PUT /api/admin/tasks/{id}/move", a.auth(a.teacher(a.adminMoveTask)))
	m.HandleFunc("GET /api/tutorials", a.auth(a.tutorials))
	m.HandleFunc("POST /api/tutorials", a.auth(a.teacher(a.saveTutorial)))
	m.HandleFunc("PUT /api/tutorials/{id}", a.auth(a.teacher(a.saveTutorial)))
	m.HandleFunc("DELETE /api/tutorials/{id}", a.auth(a.teacher(a.adminDeleteContent)))
	m.HandleFunc("GET /api/admin/users", a.auth(a.teacher(a.adminUsers)))
	m.HandleFunc("DELETE /api/admin/users/{id}", a.auth(a.teacher(a.adminDeleteUser)))
	m.HandleFunc("PUT /api/admin/users/{id}/role", a.auth(a.teacher(a.adminUserUpdate)))
	m.HandleFunc("PUT /api/admin/users/{id}/level", a.auth(a.teacher(a.adminUserUpdate)))
	m.HandleFunc("PUT /api/admin/users/{id}/team", a.auth(a.teacher(a.adminAssignTeam)))
	m.HandleFunc("GET /api/admin/teams", a.auth(a.teacher(a.adminTeams)))
	m.HandleFunc("POST /api/admin/teams", a.auth(a.teacher(a.adminTeams)))
	m.HandleFunc("PUT /api/admin/teams/{id}", a.auth(a.teacher(a.adminTeams)))
	m.HandleFunc("DELETE /api/admin/teams/{id}", a.auth(a.teacher(a.adminTeams)))
	m.HandleFunc("GET /api/admin/invite-codes", a.auth(a.teacher(a.adminInvites)))
	m.HandleFunc("POST /api/admin/invite-codes", a.auth(a.teacher(a.adminInvites)))
	m.HandleFunc("DELETE /api/admin/invite-codes/{id}", a.auth(a.teacher(a.adminInvites)))
	m.HandleFunc("GET /api/admin/submissions", a.auth(a.teacher(a.adminSubmissions)))
	m.HandleFunc("GET /api/admin/mentorship-graph", a.auth(a.teacher(a.adminGraph)))
}
func reply(w http.ResponseWriter, status int, data any, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	v := map[string]any{"code": status, "message": msg}
	if data != nil {
		v["data"] = data
	}
	_ = json.NewEncoder(w).Encode(v)
}
func ok(w http.ResponseWriter, v any)       { reply(w, 200, v, "success") }
func bad(w http.ResponseWriter, msg string) { reply(w, 400, nil, msg) }
func fail(w http.ResponseWriter)            { reply(w, 500, nil, "服务器内部错误") }
func (a *App) auth(fn func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(a.key) < 32 {
			reply(w, 503, nil, "JWT 密钥未配置")
			return
		}
		s := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if s == r.Header.Get("Authorization") {
			reply(w, 401, nil, "请先登录")
			return
		}
		c := new(claims)
		token, err := jwt.ParseWithClaims(s, c, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid algorithm")
			}
			return a.key, nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid || c.UserID <= 0 || c.Subject == "" {
			reply(w, 401, nil, "登录已过期")
			return
		}
		var found int
		if err = a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM users WHERE id=? AND username=?`, c.UserID, c.Subject).Scan(&found); err != nil {
			reply(w, 401, nil, "用户不存在")
			return
		}
		fn(w, r, c.UserID)
	}
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if len(a.key) < 32 {
		reply(w, 503, nil, "JWT 密钥未配置")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		bad(w, "用户名和密码不能为空")
		return
	}
	if len(req.Username) > 80 || len(req.Password) > 72 {
		bad(w, "用户名或密码错误")
		return
	}
	if !a.allowAttempt("login-ip:"+clientIP(r), 240) || !a.allowAttempt("login-user:"+strings.ToLower(req.Username), 20) {
		reply(w, 429, nil, "尝试过于频繁，请稍后再试")
		return
	}
	select {
	case a.bcryptSlots <- struct{}{}:
		defer func() { <-a.bcryptSlots }()
	case <-r.Context().Done():
		return
	}
	var id int64
	var password, role string
	err := a.DB.QueryRowContext(r.Context(), `SELECT id,password,COALESCE(role,'') FROM users WHERE username=?`, req.Username).Scan(&id, &password, &role)
	if err == sql.ErrNoRows {
		password = string(dummyPasswordHash)
	} else if err != nil {
		fail(w)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)) != nil || id == 0 {
		bad(w, "用户名或密码错误")
		return
	}
	token, err := a.loginToken(id, req.Username, role)
	if err != nil {
		fail(w)
		return
	}
	p, err := a.userProfile(r, id)
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"token": token, "user": p})
}
func (a *App) signingMethod() *jwt.SigningMethodHMAC {
	method := jwt.SigningMethodHS256
	if len(a.key) >= 64 {
		method = jwt.SigningMethodHS512
	} else if len(a.key) >= 48 {
		method = jwt.SigningMethodHS384
	}
	return method
}
func (a *App) loginToken(id int64, username, role string) (string, error) {
	now := time.Now()
	c := claims{UserID: id, Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: username, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour))}}
	return jwt.NewWithClaims(a.signingMethod(), c).SignedString(a.key)
}
func (a *App) profile(w http.ResponseWriter, r *http.Request, id int64) {
	p, err := a.userProfile(r, id)
	if err != nil {
		fail(w)
		return
	}
	ok(w, p)
}
func (a *App) userProfile(r *http.Request, id int64) (map[string]any, error) {
	var uid int64
	var username, role string
	var name, email, avatar, created, teamName sql.NullString
	var teamID sql.NullInt64
	var xp, level, completed int
	err := a.DB.QueryRowContext(r.Context(), `SELECT u.id,u.username,u.name,u.email,COALESCE(u.role,''),u.team_id,t.name,u.avatar,u.total_xp,COALESCE(u.level,0),u.created_at,(SELECT count(*) FROM user_level_progress p WHERE p.user_id=u.id AND p.status='COMPLETED') FROM users u LEFT JOIN teams t ON t.id=u.team_id WHERE u.id=?`, id).Scan(&uid, &username, &name, &email, &role, &teamID, &teamName, &avatar, &xp, &level, &created, &completed)
	if err != nil {
		return nil, err
	}
	title := "青铜"
	for _, v := range []struct {
		n int
		s string
	}{{10, "白银"}, {20, "黄金"}, {30, "铂金"}, {40, "钻石"}, {50, "大师"}, {60, "宗师"}, {70, "王者"}} {
		if level >= v.n {
			title = v.s
		}
	}
	return map[string]any{"id": uid, "username": username, "name": nullString(name), "email": nullString(email), "role": role, "teamId": nullInt(teamID), "teamName": nullString(teamName), "avatar": nullString(avatar), "totalXp": xp, "level": level, "levelTitle": title, "completedLevels": completed, "createdAt": a.timestamp(created)}, nil
}
func nullString(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}
func nullInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
func idParam(r *http.Request) (int64, error) {
	i, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || i <= 0 {
		return 0, errors.New("invalid id")
	}
	return i, nil
}
