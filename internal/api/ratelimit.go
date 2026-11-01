package api

import (
	"golang.org/x/crypto/bcrypt"
	"net"
	"net/http"
	"time"
)

type rateWindow struct {
	count int
	until time.Time
}

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("unusable-placeholder-password"), bcrypt.DefaultCost)

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func (a *App) allowAttempt(key string, max int) bool {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	now := time.Now()
	if len(a.rates) > 4096 {
		for k, v := range a.rates {
			if now.After(v.until) {
				delete(a.rates, k)
			}
		}
		if len(a.rates) > 8192 {
			return false
		}
	}
	v := a.rates[key]
	if now.After(v.until) {
		v = rateWindow{until: now.Add(time.Minute)}
	}
	v.count++
	a.rates[key] = v
	return v.count <= max
}
