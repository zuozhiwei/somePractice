package api

import (
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type resetEntry struct {
	code     string
	expires  time.Time
	attempts int
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	if len(a.key) < 32 {
		reply(w, 503, nil, "JWT 密钥未配置")
		return
	}
	if !a.allowAttempt("register:"+clientIP(r), 40) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return
	}
	var x struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		Name           string `json:"name"`
		Email          string `json:"email"`
		Role           string `json:"role"`
		TeamInviteCode string `json:"teamInviteCode"`
		InviteToken    string `json:"inviteToken"`
		MentorID       *int64 `json:"mentorId"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if json.NewDecoder(r.Body).Decode(&x) != nil {
		bad(w, "请求无效")
		return
	}
	x.Username = strings.TrimSpace(x.Username)
	x.Email = strings.TrimSpace(x.Email)
	x.Role = strings.ToUpper(x.Role)
	if len(x.Username) < 1 || len(x.Username) > 80 || len(x.Password) < 6 || len(x.Password) > 72 || len(x.Name) > 100 || len(x.Email) > 254 || (x.Role != "INTERNAL" && x.Role != "EXTERNAL") {
		bad(w, "注册信息无效")
		return
	}
	if x.Email != "" {
		if addr, err := mail.ParseAddress(x.Email); err != nil || addr.Address != x.Email {
			bad(w, "邮箱无效")
			return
		}
	}
	var mentor *int64
	if x.InviteToken != "" {
		c := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(x.InviteToken, c, func(t *jwt.Token) (any, error) { return a.key, nil }, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			bad(w, "邀请链接无效或已过期")
			return
		}
		if _, isLogin := c["userId"]; isLogin {
			bad(w, "邀请链接无效")
			return
		}
		subject, _ := c.GetSubject()
		id, err := strconv.ParseInt(subject, 10, 64)
		if err != nil || id <= 0 {
			bad(w, "邀请链接无效")
			return
		}
		mentor = &id
		x.Role = "INTERNAL"
	} else if x.MentorID != nil {
		bad(w, "请使用有效的邀请链接")
		return
	}
	select {
	case a.bcryptSlots <- struct{}{}:
		defer func() { <-a.bcryptSlots }()
	case <-r.Context().Done():
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(x.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w)
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow(`SELECT count(*) FROM users WHERE username=? OR (?<>'' AND email=?)`, x.Username, x.Email, x.Email).Scan(&count); err != nil {
		fail(w)
		return
	}
	if count > 0 {
		bad(w, "用户名或邮箱已存在")
		return
	}
	var team sql.NullInt64
	if x.Role == "INTERNAL" && mentor == nil {
		err = tx.QueryRow(`SELECT team_id FROM invite_codes WHERE code=? AND active=1 AND expires_at>?`, x.TeamInviteCode, time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")).Scan(&team)
		if err != nil {
			bad(w, "邀请码无效或已过期")
			return
		}
	}
	if mentor != nil {
		if err = tx.QueryRow(`SELECT count(*) FROM users WHERE id=? AND role IN ('INTERNAL','TEACHER')`, *mentor).Scan(&count); err != nil || count == 0 {
			bad(w, "邀请的师父不存在")
			return
		}
	}
	var email any
	if x.Email != "" {
		email = x.Email
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	res, err := tx.Exec(`INSERT INTO users(username,password,name,email,role,team_id,total_xp,level,created_at) VALUES(?,?,?,?,?,?,0,0,?)`, x.Username, string(hash), x.Name, email, x.Role, nullInt(team), now)
	if err != nil {
		bad(w, "注册失败：用户名或邮箱已存在")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		fail(w)
		return
	}
	if mentor != nil {
		if _, err = tx.Exec(`INSERT INTO mentorships(mentor_id,apprentice_id,status,created_at) VALUES(?,?,'ACTIVE',?)`, *mentor, id, now); err != nil {
			fail(w)
			return
		}
	}
	if team.Valid {
		if _, err = tx.Exec(`INSERT INTO user_teams(team_id,user_id,role,joined_at) VALUES(?,?,'MEMBER',?)`, team.Int64, id, now); err != nil {
			fail(w)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	token, err := a.loginToken(id, x.Username, x.Role)
	if err != nil {
		fail(w)
		return
	}
	profile, err := a.userProfile(r, id)
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"token": token, "user": profile})
}
func (a *App) updateProfile(w http.ResponseWriter, r *http.Request, id int64) {
	var x struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&x) != nil || len(x.Name) > 100 || len(x.Email) > 254 {
		bad(w, "资料无效")
		return
	}
	if x.Email != "" {
		if addr, err := mail.ParseAddress(x.Email); err != nil || addr.Address != x.Email {
			bad(w, "邮箱无效")
			return
		}
	}
	_, err := a.DB.ExecContext(r.Context(), `UPDATE users SET name=CASE WHEN ?<>'' THEN ? ELSE name END,email=CASE WHEN ?<>'' THEN ? ELSE email END WHERE id=?`, x.Name, x.Name, x.Email, x.Email, id)
	if err != nil {
		bad(w, "邮箱已被注册")
		return
	}
	a.profile(w, r, id)
}
func (a *App) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !a.allowAttempt("reset-ip:"+clientIP(r), 100) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return
	}
	var x struct {
		Email string `json:"email"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&x) != nil || len(x.Email) > 254 {
		bad(w, "邮箱无效")
		return
	}
	address, err := mail.ParseAddress(x.Email)
	if err != nil || address.Address != x.Email {
		bad(w, "邮箱无效")
		return
	}
	host, user, pass := os.Getenv("SMTP_HOST"), os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD")
	if host == "" || user == "" || pass == "" {
		reply(w, 503, nil, "邮件服务未配置")
		return
	}
	var n int
	if err := a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM users WHERE email=?`, x.Email).Scan(&n); err != nil {
		fail(w)
		return
	}
	if n == 0 {
		ok(w, nil)
		return
	}
	a.resetMu.Lock()
	prev, exists := a.resets[x.Email]
	if exists && time.Until(prev.expires) > 4*time.Minute {
		a.resetMu.Unlock()
		reply(w, 429, nil, "请求过于频繁")
		return
	}
	num, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		a.resetMu.Unlock()
		fail(w)
		return
	}
	code := fmt.Sprintf("%06d", num.Int64())
	a.resets[x.Email] = resetEntry{code: code, expires: time.Now().Add(5 * time.Minute)}
	a.resetMu.Unlock()
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	err = sendResetMail(host, port, user, pass, x.Email, code, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		a.resetMu.Lock()
		delete(a.resets, x.Email)
		a.resetMu.Unlock()
		reply(w, 502, nil, "邮件发送失败")
		return
	}
	ok(w, nil)
}

func sendResetMail(host, port, user, pass, email, code string, tlsConfig *tls.Config) error {
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	var c *smtp.Client
	if err == nil {
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
		c, err = smtp.NewClient(conn, host)
		if err != nil {
			_ = conn.Close()
		}
	}
	if err == nil {
		defer c.Close()
		if ok, _ := c.Extension("STARTTLS"); !ok {
			err = fmt.Errorf("STARTTLS unavailable")
		} else {
			err = c.StartTLS(tlsConfig)
		}
		if err == nil {
			err = c.Auth(smtp.PlainAuth("", user, pass, host))
		}
		if err == nil {
			err = c.Mail(user)
		}
		if err == nil {
			err = c.Rcpt(email)
		}
		if err == nil {
			var wr interface {
				Write([]byte) (int, error)
				Close() error
			}
			wr, err = c.Data()
			if err == nil {
				_, err = wr.Write([]byte("From: " + user + "\r\nTo: " + email + "\r\nSubject: Skilltree password reset\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nCode: " + code + " (valid for 5 minutes)\r\n"))
				if e := wr.Close(); err == nil {
					err = e
				}
			}
		}
		if err == nil {
			err = c.Quit()
		}
	}
	return err
}
func (a *App) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !a.allowAttempt("reset-confirm:"+clientIP(r), 100) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return
	}
	var x struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"newPassword"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&x) != nil || len(x.Password) < 6 || len(x.Password) > 72 {
		bad(w, "请求无效")
		return
	}
	a.resetMu.Lock()
	entry, okCode := a.resets[x.Email]
	if !okCode || time.Now().After(entry.expires) || entry.attempts >= 5 {
		delete(a.resets, x.Email)
		a.resetMu.Unlock()
		bad(w, "验证码无效或已过期")
		return
	}
	if subtle.ConstantTimeCompare([]byte(entry.code), []byte(x.Code)) != 1 {
		entry.attempts++
		a.resets[x.Email] = entry
		a.resetMu.Unlock()
		bad(w, "验证码错误")
		return
	}
	delete(a.resets, x.Email)
	a.resetMu.Unlock()
	hash, err := bcrypt.GenerateFromPassword([]byte(x.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w)
		return
	}
	res, err := a.DB.ExecContext(r.Context(), `UPDATE users SET password=? WHERE email=?`, string(hash), x.Email)
	if err != nil {
		fail(w)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		bad(w, "用户不存在")
		return
	}
	ok(w, nil)
}
