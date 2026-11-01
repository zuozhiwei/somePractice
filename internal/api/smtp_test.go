package api

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSMTPStartTLSAndPasswordReset(t *testing.T) {
	fixture := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer fixture.Close()
	pool := x509.NewCertPool()
	pool.AddCert(fixture.Certificate())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	message := make(chan string, 1)
	serverErr := make(chan error, 1)
	go func() {
		raw, e := ln.Accept()
		if e != nil {
			serverErr <- e
			return
		}
		defer raw.Close()
		_ = raw.SetDeadline(time.Now().Add(5 * time.Second))
		conn := net.Conn(raw)
		reader := bufio.NewReader(conn)
		write := func(s string) error { _, e := fmt.Fprint(conn, s); return e }
		if e = write("220 localhost ESMTP\r\n"); e != nil {
			serverErr <- e
			return
		}
		tlsActive := false
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				serverErr <- e
				return
			}
			upper := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
				if tlsActive {
					e = write("250-localhost\r\n250 AUTH PLAIN\r\n")
				} else {
					e = write("250-localhost\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
				}
			case strings.HasPrefix(upper, "STARTTLS"):
				if e = write("220 ready\r\n"); e == nil {
					secured := tls.Server(conn, fixture.TLS)
					e = secured.Handshake()
					if e == nil {
						conn = secured
						reader = bufio.NewReader(conn)
						tlsActive = true
					}
				}
			case strings.HasPrefix(upper, "AUTH "):
				e = write("235 authenticated\r\n")
			case strings.HasPrefix(upper, "MAIL "), strings.HasPrefix(upper, "RCPT "):
				e = write("250 ok\r\n")
			case upper == "DATA":
				if e = write("354 end with dot\r\n"); e == nil {
					var lines []string
					for {
						var l string
						l, e = reader.ReadString('\n')
						if e != nil {
							break
						}
						if l == ".\r\n" {
							break
						}
						lines = append(lines, l)
					}
					if e == nil {
						message <- strings.Join(lines, "")
						e = write("250 queued\r\n")
					}
				}
			case upper == "QUIT":
				_ = write("221 bye\r\n")
				serverErr <- nil
				return
			default:
				e = fmt.Errorf("unexpected SMTP command: %s", upper)
			}
			if e != nil {
				serverErr <- e
				return
			}
		}
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	if err = sendResetMail(host, port, "sender@example.test", "password", "student@example.test", "123456", &tls.Config{RootCAs: pool, ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		t.Fatal(err)
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
	if msg := <-message; !strings.Contains(msg, "123456") {
		t.Fatal("missing code")
	}
	db, h := testApp(t)
	if _, err = db.Exec(`UPDATE users SET email='student@example.test' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	// Exercise reset code expiration, validation, and one-time use without sending a real email.
	app := New(db)
	app.resets["student@example.test"] = resetEntry{code: "123456", expires: time.Now().Add(time.Minute)}
	mux := http.NewServeMux()
	app.Register(mux)
	w := call(mux, "POST", "/api/auth/reset-password", `{"email":"student@example.test","code":"123456","newPassword":"new-password"}`, "")
	if w.Code != 200 {
		t.Fatalf("reset %d %s", w.Code, w.Body)
	}
	w = call(h, "POST", "/api/auth/login", `{"username":"alice","password":"new-password"}`, "")
	if w.Code != 200 {
		t.Fatalf("login after reset %d %s", w.Code, w.Body)
	}
	w = call(mux, "POST", "/api/auth/reset-password", `{"email":"student@example.test","code":"123456","newPassword":"other-password"}`, "")
	if w.Code != 400 {
		t.Fatalf("reset code reused %d", w.Code)
	}
}
