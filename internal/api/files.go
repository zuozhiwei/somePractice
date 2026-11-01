package api

import (
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func qiniuConfigured() bool {
	return os.Getenv("QINIU_ACCESS_KEY") != "" && os.Getenv("QINIU_SECRET_KEY") != "" && os.Getenv("QINIU_BUCKET") != "" && os.Getenv("QINIU_DOMAIN") != ""
}
func qiniuSign(value string) string {
	h := hmac.New(sha1.New, []byte(os.Getenv("QINIU_SECRET_KEY")))
	_, _ = h.Write([]byte(value))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}
func (a *App) uploadToken(w http.ResponseWriter, r *http.Request, user int64) {
	if !qiniuConfigured() {
		reply(w, 503, nil, "七牛云未配置")
		return
	}
	prefix := fmt.Sprintf("uploads/%d/", user)
	data, _ := json.Marshal(map[string]any{"scope": os.Getenv("QINIU_BUCKET") + ":" + prefix, "isPrefixalScope": 1, "insertOnly": 1, "fsizeLimit": 10485760, "deadline": time.Now().Unix() + 3600, "returnBody": `{"key":"$(key)","hash":"$(etag)","mimeType":"$(mimeType)","size":$(fsize)}`})
	encoded := base64.URLEncoding.EncodeToString(data)
	token := os.Getenv("QINIU_ACCESS_KEY") + ":" + qiniuSign(encoded) + ":" + encoded
	host := os.Getenv("QINIU_UPLOAD_HOST")
	if host == "" {
		host = "https://upload-z1.qiniup.com"
	}
	ok(w, map[string]any{"token": token, "uploadHost": host, "domain": strings.TrimRight(os.Getenv("QINIU_DOMAIN"), "/"), "expiresIn": 3600, "keyPrefix": prefix})
}
func signURL(raw string) string {
	if !qiniuConfigured() {
		return raw
	}
	base := strings.TrimRight(os.Getenv("QINIU_DOMAIN"), "/")
	if !strings.HasPrefix(raw, base+"/") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Query().Has("token") {
		return raw
	}
	sep := "?"
	if u.RawQuery != "" {
		sep = "&"
	}
	toSign := raw + sep + fmt.Sprintf("e=%d", time.Now().Unix()+3600)
	return toSign + "&token=" + os.Getenv("QINIU_ACCESS_KEY") + ":" + qiniuSign(toSign)
}
func (a *App) signURLs(w http.ResponseWriter, r *http.Request, user int64) {
	var body struct {
		URLs []string `json:"urls"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.URLs) > 200 {
		bad(w, "链接数量或内容无效")
		return
	}
	result := map[string]string{}
	prefix := strings.TrimRight(os.Getenv("QINIU_DOMAIN"), "/") + fmt.Sprintf("/uploads/%d/", user)
	for _, v := range body.URLs {
		result[v] = v
		if qiniuConfigured() && strings.HasPrefix(v, prefix) {
			u, err := url.Parse(v)
			if err == nil && !strings.Contains(u.EscapedPath(), "%2f") && !strings.Contains(u.Path, "/../") {
				result[v] = signURL(v)
			}
		}
	}
	ok(w, result)
}

var markdownURL = regexp.MustCompile(`https?://[^\s\)"'<>]+`)

func signMarkdown(s string) string { return markdownURL.ReplaceAllStringFunc(s, signURL) }
func signedContent(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return signMarkdown(s.String)
}
func normalizeContent(s string) string {
	base := strings.TrimRight(os.Getenv("QINIU_DOMAIN"), "/")
	if base == "" {
		return s
	}
	return markdownURL.ReplaceAllStringFunc(s, func(raw string) string {
		if !strings.HasPrefix(raw, base+"/") {
			return raw
		}
		u, err := url.Parse(raw)
		if err != nil {
			return raw
		}
		q := u.Query()
		q.Del("token")
		q.Del("e")
		u.RawQuery = q.Encode()
		return u.String()
	})
}
