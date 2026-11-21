package api

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skilltree-go/internal/curriculum"
)

func TestLearningCompressionPreservesLargeJSON(t *testing.T) {
	db, app := testApp(t)
	token := loginToken(t, app)
	learningSQL(t, db, "DELETE FROM tutorials; DELETE FROM tasks; DELETE FROM levels; DELETE FROM maps")
	pack, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	request := func(path, encoding string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Accept-Encoding", encoding)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d, %v", path, w.Code, w.Header())
		}
		return w
	}

	const lesson = "/api/levels/1"
	original := request(lesson, "gzip;q=0")
	if original.Header().Get("Content-Encoding") != "" || original.Body.Len() < 1024 {
		t.Fatalf("expected a large identity response: %d bytes, %v", original.Body.Len(), original.Header())
	}
	compressed := request(lesson, "gzip")
	if compressed.Header().Get("Content-Encoding") != "gzip" ||
		compressed.Header().Get("Vary") != "Accept-Encoding" || compressed.Body.Len() >= original.Body.Len() {
		t.Fatalf("expected a smaller gzip response: %d bytes, %v", compressed.Body.Len(), compressed.Header())
	}
	t.Logf("published lesson: identity %d bytes, gzip %d bytes", original.Body.Len(), compressed.Body.Len())
	reader, err := gzip.NewReader(compressed.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(decoded, original.Body.Bytes()) {
		t.Fatalf("gzip changed the response: %v", err)
	}

	profile := request("/api/auth/profile", "gzip")
	if profile.Header().Get("Content-Encoding") != "" || profile.Header().Get("Vary") == "Accept-Encoding" {
		t.Fatalf("profile response entered the course compression path: %v", profile.Header())
	}
}

func TestLearningValidatorsTrackCurrentUserAndReview(t *testing.T) {
	db, app := testApp(t)
	alice := loginToken(t, app)
	learningSQL(t, db, `
		DELETE FROM tutorials; DELETE FROM tasks; DELETE FROM levels; DELETE FROM maps;
		INSERT INTO users(id,username,password,role,total_xp,level)
		SELECT 2,'bob',password,'EXTERNAL',0,0 FROM users WHERE id=1`)
	var auth struct{ Token string }
	learningResponse(t, app, http.MethodPost, "/api/auth/login", `{"username":"bob","password":"correct-password"}`, "", &auth)
	pack, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	get := func(path, token, validator string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		r.Header.Set("Accept-Encoding", "gzip")
		r.Header.Set("If-None-Match", validator)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: missing privacy header on %d", path, w.Code)
		}
		return w
	}

	tags := map[string]string{}
	for _, path := range []string{"/api/levels/1", "/api/maps/atlas"} {
		initial := get(path, alice, "")
		tag := initial.Header().Get("ETag")
		if initial.Code != 200 || !strings.HasPrefix(tag, `W/"`) {
			t.Fatalf("%s: initial status %d, tag %q", path, initial.Code, tag)
		}
		tags[path] = tag
		same := get(path, alice, `"another-validator", `+strings.TrimPrefix(tag, "W/"))
		if same.Code != 304 || same.Body.Len() != 0 || same.Header().Get("ETag") != tag ||
			same.Header().Get("Content-Encoding") != "" || same.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatalf("%s: unchanged response %d, %d bytes, %v", path, same.Code, same.Body.Len(), same.Header())
		}
		if w := get(path, "", "*"); w.Code != 401 || w.Header().Get("ETag") != "" {
			t.Fatalf("%s: validator bypassed authorization: %d, %v", path, w.Code, w.Header())
		}
		if w := get(path, auth.Token, tag); w.Code != 200 || w.Header().Get("ETag") == tag {
			t.Fatalf("%s: reused another user's validator: %d, %v", path, w.Code, w.Header())
		}
	}

	learningSQL(t, db, `INSERT INTO user_task_submissions(user_id,task_id,status,ai_review_count)
		VALUES (1,2,'APPROVED',0)`)
	for path, tag := range tags {
		w := get(path, alice, tag)
		if w.Code != 200 || w.Body.Len() == 0 || w.Header().Get("ETag") == tag {
			t.Fatalf("%s: review change did not produce a new response: %d, %v", path, w.Code, w.Header())
		}
	}
}
