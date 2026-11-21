package frontend

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticCachingAndCompression(t *testing.T) {
	js := []byte(strings.Repeat("export const title = '校园树洞';\n", 120))
	html := []byte("<!doctype html><html><body>" + strings.Repeat("<p>校园树洞</p>", 80) + "</body></html>")
	image := []byte(strings.Repeat("webp", 1200))
	h := handler(fstest.MapFS{
		"assets/entry-fC1_4tLZ.js":   {Data: js},
		"assets/cover-j7Zme_41.webp": {Data: image},
		"assets/settings.json":       {Data: []byte(`{"version":1}`)},
		"index.html":                 {Data: html},
	})
	request := func(method, url string, headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, url, nil)
		for name, value := range headers {
			r.Header.Set(name, value)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	const script = "/assets/entry-fC1_4tLZ.js"
	identity := request("GET", script, map[string]string{"Accept-Encoding": "gzip;q=0, *;q=1"})
	if identity.Code != 200 || identity.Header().Get("Content-Encoding") != "" ||
		identity.Header().Get("Vary") != "Accept-Encoding" || !bytes.Equal(identity.Body.Bytes(), js) {
		t.Fatalf("identity script response: %d, %v", identity.Code, identity.Header())
	}
	compressed := request("GET", script, map[string]string{"Accept-Encoding": "br, GZip;q=0.8"})
	gzipTag := compressed.Header().Get("ETag")
	if compressed.Code != 200 || compressed.Header().Get("Content-Encoding") != "gzip" ||
		!strings.Contains(compressed.Header().Get("Content-Type"), "javascript") ||
		compressed.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		gzipTag == "" || identity.Header().Get("ETag") == gzipTag || compressed.Body.Len() >= len(js) {
		t.Fatalf("compressed script response: %d, %v, %d bytes", compressed.Code, compressed.Header(), compressed.Body.Len())
	}
	reader, err := gzip.NewReader(compressed.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(decoded, js) {
		t.Fatalf("compressed response changed JavaScript: %v", err)
	}
	head := request("HEAD", script, map[string]string{"Accept-Encoding": "*"})
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("ETag") != gzipTag ||
		head.Header().Get("Content-Length") != compressed.Header().Get("Content-Length") {
		t.Fatalf("compressed script HEAD response: %d, %v", head.Code, head.Header())
	}
	cached := request("GET", script, map[string]string{"Accept-Encoding": "gzip", "If-None-Match": gzipTag})
	if cached.Code != 304 || cached.Body.Len() != 0 || cached.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("conditional request: %d, %v", cached.Code, cached.Header())
	}
	ranged := request("GET", script, map[string]string{"Accept-Encoding": "gzip", "Range": "bytes=0-9"})
	if ranged.Code != 206 || ranged.Header().Get("Content-Encoding") != "" || !bytes.Equal(ranged.Body.Bytes(), js[:10]) {
		t.Fatalf("range request: %d, %v, %q", ranged.Code, ranged.Header(), ranged.Body.String())
	}
	preferred := request("GET", script, map[string]string{"Accept-Encoding": "identity;q=1, gzip;q=0.4"})
	if preferred.Header().Get("Content-Encoding") != "" || !bytes.Equal(preferred.Body.Bytes(), js) {
		t.Fatalf("preferred identity response: %d, %v", preferred.Code, preferred.Header())
	}
	for _, url := range []string{"/", "/learn", "/assets/settings.json"} {
		result := request(http.MethodGet, url, nil)
		if result.Code != 200 || result.Header().Get("Cache-Control") != "no-cache" || result.Header().Get("ETag") == "" {
			t.Fatalf("revalidated %s: %d, %v", url, result.Code, result.Header())
		}
		stillCurrent := request("GET", url, map[string]string{"If-None-Match": result.Header().Get("ETag")})
		if stillCurrent.Code != 304 || stillCurrent.Body.Len() != 0 {
			t.Fatalf("conditional %s: %d %v", url, stillCurrent.Code, stillCurrent.Header())
		}
	}
	asset := request("GET", "/assets/cover-j7Zme_41.webp", map[string]string{"Accept-Encoding": "gzip"})
	if asset.Header().Get("Content-Encoding") != "" || !bytes.Equal(asset.Body.Bytes(), image) {
		t.Fatalf("image response: %d, %v", asset.Code, asset.Header())
	}
	missing := request("GET", "/assets/gone-j7Zme_41.js", nil)
	if missing.Code != 404 || missing.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("missing hash: %d, %v", missing.Code, missing.Header())
	}
}
