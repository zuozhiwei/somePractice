package api

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

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
	var mentorVersion int64
	if x.InviteToken != "" {
		c := new(mentorInviteClaims)
		token, err := jwt.ParseWithClaims(x.InviteToken, c, func(t *jwt.Token) (any, error) { return a.key, nil }, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			bad(w, "邀请链接无效或已过期")
			return
		}
		if c.LoginUserID != nil {
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
		mentorVersion = c.Version
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
	if err = tx.QueryRow(`SELECT count(*) FROM users WHERE username=? OR (?<>'' AND lower(trim(email))=lower(?))`, x.Username, x.Email, x.Email).Scan(&count); err != nil {
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
		if err = tx.QueryRow(`SELECT count(*) FROM users u LEFT JOIN login_versions v ON v.user_id=u.id
			WHERE u.id=? AND COALESCE(v.version,0)=? AND u.role IN ('INTERNAL','TEACHER','SUPERADMIN') AND COALESCE(u.level,0)>0`, *mentor, mentorVersion).Scan(&count); err != nil || count == 0 {
			bad(w, "邀请链接已失效，请联系师父重新获取")
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
	var version int64
	if err = tx.QueryRowContext(r.Context(), "SELECT version FROM login_versions WHERE user_id=?", id).Scan(&version); err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	token, err := a.loginToken(id, x.Username, x.Role, version)
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
	x.Email = strings.TrimSpace(x.Email)
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
func sendResetMail(host, port, user, pass, email, code string, tlsConfig *tls.Config, implicitTLS bool) error {
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return err
	}
	if implicitTLS {
		conn = tls.Client(conn, tlsConfig)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if !implicitTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("STARTTLS unavailable")
		}
		if err = c.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if err = c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		return err
	}
	if err = c.Mail(user); err != nil {
		return err
	}
	if err = c.Rcpt(email); err != nil {
		return err
	}
	wr, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(wr, "From: %s\r\nTo: %s\r\nDate: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		user, email, time.Now().Format(time.RFC1123Z), mime.QEncoding.Encode("UTF-8", "技能树学习系统 · 找回密码验证码"))
	if err != nil {
		return err
	}
	body := quotedprintable.NewWriter(wr)
	_, err = fmt.Fprintf(body, "你正在找回技能树学习系统的登录密码。你的验证码是\r\n\r\n%s\r\n\r\n验证码 5 分钟有效。若非本人申请，可忽略此邮件。\r\n", code)
	if err != nil {
		return err
	}
	if err = body.Close(); err != nil {
		return err
	}
	if err = wr.Close(); err != nil {
		return err
	}
	_ = c.Quit()
	return nil
}
