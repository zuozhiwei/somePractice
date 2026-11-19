package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const invalidRecovery = "账号或验证码不正确，或验证码已失效"
const emailRequestMessage = "若账号绑定了可用邮箱，验证码将发到该邮箱。请检查收件箱和垃圾邮件。"

var errMailUnconfigured = errors.New("邮件服务尚未配置")
var errEmailCooldown = errors.New("请在一分钟后重新获取验证码")

type recoveryAccount struct {
	id                              int64
	username, email, password, role string
	emailOwners                     int
}

type recoveryLookup interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const accountForRecovery = `SELECT u.id,u.username,COALESCE(u.email,''),u.password,COALESCE(u.role,''),
	(SELECT count(*) FROM users e WHERE lower(trim(e.email))=lower(trim(u.email)))
	FROM users u`

func recoveryByID(ctx context.Context, db recoveryLookup, id int64) (recoveryAccount, error) {
	var u recoveryAccount
	err := db.QueryRowContext(ctx, accountForRecovery+" WHERE u.id=?", id).Scan(&u.id, &u.username, &u.email, &u.password, &u.role, &u.emailOwners)
	return u, err
}

func lookupRecovery(ctx context.Context, db recoveryLookup, identifier, method string) (recoveryAccount, error) {
	var id int64
	if method == "admin" {
		err := db.QueryRowContext(ctx, "SELECT id FROM users WHERE username=?", identifier).Scan(&id)
		if err != nil {
			return recoveryAccount{}, err
		}
	} else {
		var count int
		err := db.QueryRowContext(ctx, `SELECT count(*), COALESCE(min(id),0) FROM users
			WHERE username=? OR lower(trim(email))=lower(?)`, identifier, identifier).Scan(&count, &id)
		if err != nil {
			return recoveryAccount{}, err
		}
		if count != 1 {
			return recoveryAccount{}, sql.ErrNoRows
		}
	}
	return recoveryByID(ctx, db, id)
}

func (u recoveryAccount) deliverable() bool {
	address := strings.TrimSpace(u.email)
	parsed, err := mail.ParseAddress(address)
	return u.emailOwners == 1 && err == nil && parsed.Address == address
}

// Digests are keyed so a copied database cannot be used to guess six-digit email codes.
func (a *App) recoveryDigest(u recoveryAccount, method, code string) []byte {
	mac := hmac.New(sha256.New, a.key)
	fmt.Fprintf(mac, "skilltree-recovery-v1:%d:%q:%q:%q:%q:%q:%q", u.id, u.username, u.password, u.email, u.role, method, code)
	return mac.Sum(nil)
}

// Insert against the same account snapshot that was checked before issuance.
func (a *App) storeRecovery(ctx context.Context, u recoveryAccount, method, code string, ttl time.Duration) error {
	now := time.Now()
	res, err := a.DB.ExecContext(ctx, `INSERT INTO password_recovery(user_id,method,digest,issued_at,expires_at,attempts_left)
		SELECT id,?,?,?,?,5 FROM users WHERE id=? AND username=? AND password=? AND coalesce(email,'')=? AND coalesce(role,'')=?
		ON CONFLICT(user_id,method) DO UPDATE SET digest=excluded.digest, issued_at=excluded.issued_at,
		expires_at=excluded.expires_at, attempts_left=5
		WHERE excluded.method='admin' OR password_recovery.issued_at<=excluded.issued_at-60`,
		method, a.recoveryDigest(u, method, code), now.Unix(), now.Add(ttl).Unix(), u.id, u.username, u.password, u.email, u.role)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errEmailCooldown
	}
	return nil
}

type recoveryRequest struct {
	Identifier  string `json:"identifier"`
	Email       string `json:"email"`
	Method      string `json:"method"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

func readRecovery(w http.ResponseWriter, r *http.Request) (recoveryRequest, bool) {
	var req recoveryRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		bad(w, "请求无效")
		return req, false
	}
	if req.Identifier == "" {
		req.Identifier = req.Email
	}
	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" || len(req.Identifier) > 254 {
		bad(w, "请输入登录用户名或绑定邮箱")
		return req, false
	}
	if req.Method == "" {
		req.Method = "email"
	}
	if req.Method != "email" && req.Method != "admin" {
		bad(w, "找回方式无效")
		return req, false
	}
	return req, true
}

func (a *App) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !a.allowAttempt("reset-ip:"+clientIP(r), 100) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return
	}
	req, valid := readRecovery(w, r)
	if !valid {
		return
	}
	u, err := lookupRecovery(r.Context(), a.DB, req.Identifier, "email")
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !u.deliverable()) {
		reply(w, 200, nil, emailRequestMessage)
		return
	}
	if err != nil {
		fail(w)
		return
	}
	a.sendRecoveryEmail(w, r, u, false)
}

func (a *App) adminRecoveryTarget(w http.ResponseWriter, r *http.Request, actor int64) (recoveryAccount, bool) {
	if !a.allowAttempt("admin-reset:"+strconv.FormatInt(actor, 10), 30) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return recoveryAccount{}, false
	}
	id, err := idParam(r)
	if err != nil {
		bad(w, "用户无效")
		return recoveryAccount{}, false
	}
	u, err := recoveryByID(r.Context(), a.DB, id)
	if errors.Is(err, sql.ErrNoRows) {
		reply(w, 404, nil, "用户不存在")
		return u, false
	}
	if err != nil {
		fail(w)
		return u, false
	}
	if u.role == "SUPERADMIN" {
		reply(w, 403, nil, "超级管理员请使用本人绑定邮箱找回密码")
		return u, false
	}
	return u, true
}

func (a *App) adminPasswordReset(w http.ResponseWriter, r *http.Request, actor int64) {
	u, ok := a.adminRecoveryTarget(w, r, actor)
	if !ok {
		return
	}
	if !u.deliverable() {
		bad(w, "绑定邮箱无效或多个账号使用了同一邮箱，请核实身份后使用管理员恢复码")
		return
	}
	a.sendRecoveryEmail(w, r, u, true)
}

func (a *App) sendRecoveryEmail(w http.ResponseWriter, r *http.Request, u recoveryAccount, admin bool) {
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		fail(w)
		return
	}
	code := fmt.Sprintf("%06d", number.Int64())
	err = a.storeRecovery(r.Context(), u, "email", code, 5*time.Minute)
	if errors.Is(err, errEmailCooldown) {
		if admin {
			reply(w, 429, nil, err.Error())
		} else {
			reply(w, 200, nil, emailRequestMessage)
		}
		return
	}
	if err != nil {
		fail(w)
		return
	}
	if err = a.sendRecoveryMail(strings.TrimSpace(u.email), code); err != nil {
		// A disconnected request must also release the resend cooldown on delivery failure.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = a.DB.ExecContext(ctx, `DELETE FROM password_recovery WHERE user_id=? AND method='email' AND digest=?`, u.id, a.recoveryDigest(u, "email", code))
		if errors.Is(err, errMailUnconfigured) {
			reply(w, 503, nil, "邮件服务暂不可用，请联系网站管理员索取恢复码")
		} else {
			reply(w, 502, nil, "邮件投递失败，请稍后重试或联系网站管理员索取恢复码")
		}
		return
	}
	if admin {
		ok(w, nil)
	} else {
		reply(w, 200, nil, emailRequestMessage)
	}
}

func deliverResetMail(to, code string) error {
	host, user, pass := os.Getenv("SMTP_HOST"), os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD")
	if host == "" || user == "" || pass == "" {
		return errMailUnconfigured
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendResetMail(host, port, user, pass, to, code, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}, port == "465")
}

func (a *App) adminRecoveryCode(w http.ResponseWriter, r *http.Request, actor int64) {
	w.Header().Set("Cache-Control", "no-store")
	u, okTarget := a.adminRecoveryTarget(w, r, actor)
	if !okTarget {
		return
	}
	var x struct {
		Verified bool `json:"identityVerified"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if json.NewDecoder(r.Body).Decode(&x) != nil || !x.Verified {
		bad(w, "请先核实用户身份")
		return
	}
	var secret [10]byte
	if _, err := rand.Read(secret[:]); err != nil {
		fail(w)
		return
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret[:])
	if err := a.storeRecovery(r.Context(), u, "admin", code, 15*time.Minute); err != nil {
		fail(w)
		return
	}
	formatted := code[:4] + "-" + code[4:8] + "-" + code[8:12] + "-" + code[12:]
	ok(w, map[string]any{"username": u.username, "recoveryCode": formatted, "validMinutes": 15})
}

func (a *App) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !a.allowAttempt("reset-confirm:"+clientIP(r), 100) {
		reply(w, 429, nil, "请求过于频繁，请稍后再试")
		return
	}
	req, valid := readRecovery(w, r)
	if !valid {
		return
	}
	if len(req.NewPassword) < 6 || len(req.NewPassword) > 72 {
		bad(w, "密码长度需为 6 至 72 字节")
		return
	}
	code := strings.TrimSpace(req.Code)
	if req.Method == "admin" {
		code = strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	}
	ctx := r.Context()
	u, err := lookupRecovery(ctx, a.DB, req.Identifier, req.Method)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && ((req.Method == "email" && !u.deliverable()) || (req.Method == "admin" && u.role == "SUPERADMIN"))) {
		bad(w, invalidRecovery)
		return
	}
	if err != nil {
		fail(w)
		return
	}
	digest := a.recoveryDigest(u, req.Method, code)
	var saved []byte
	err = a.DB.QueryRowContext(ctx, `SELECT digest FROM password_recovery WHERE user_id=? AND method=? AND expires_at>? AND attempts_left>0`, u.id, req.Method, time.Now().Unix()).Scan(&saved)
	if errors.Is(err, sql.ErrNoRows) {
		bad(w, invalidRecovery)
		return
	}
	if err != nil {
		fail(w)
		return
	}
	if !hmac.Equal(digest, saved) {
		_, err = a.DB.ExecContext(ctx, `UPDATE password_recovery SET attempts_left=attempts_left-1 WHERE user_id=? AND method=? AND digest=? AND attempts_left>0`, u.id, req.Method, saved)
		if err != nil {
			fail(w)
			return
		}
		bad(w, invalidRecovery)
		return
	}
	select {
	case a.bcryptSlots <- struct{}{}:
		defer func() { <-a.bcryptSlots }()
	case <-ctx.Done():
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fail(w)
		return
	}
	// The credential check, password update and trigger-based revocation share one SQLite statement.
	res, err := a.DB.ExecContext(ctx, `UPDATE users SET password=? WHERE id=? AND password=? AND username=? AND COALESCE(email,'')=? AND COALESCE(role,'')=?
		AND (?='admin' OR 1=(SELECT count(*) FROM users e WHERE lower(trim(e.email))=lower(trim(?))))
		AND EXISTS(SELECT 1 FROM password_recovery WHERE user_id=? AND method=? AND digest=? AND attempts_left>0 AND expires_at>?)`,
		string(hash), u.id, u.password, u.username, u.email, u.role, req.Method, u.email, u.id, req.Method, digest, time.Now().Unix())
	if err != nil {
		fail(w)
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		fail(w)
		return
	}
	if n != 1 {
		bad(w, invalidRecovery)
		return
	}
	a.closeUserChats(u.id)
	ok(w, nil)
}
