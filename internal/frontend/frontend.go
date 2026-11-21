package frontend

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"skilltree-go/internal/transfer"
)

//go:embed all:dist
var assets embed.FS

var hashedAsset = regexp.MustCompile(`-[A-Za-z0-9_-]{8}\.[a-zA-Z0-9]+$`)

type representations struct {
	etag, compressedETag string
	mediaType            string
	compressed           []byte
}

func Handler() http.Handler {
	root, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return handler(root)
}

func handler(root fs.FS) http.Handler {
	variants := make(map[string]representations)
	encoder := gzip.NewWriter(io.Discard)
	if err := fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.Type().IsRegular() {
			return err
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		item := representations{
			etag:      etag(data),
			mediaType: mime.TypeByExtension(path.Ext(name)),
		}
		if len(data) >= 512 && item.mediaType != "" && compressible(name) {
			var buffer bytes.Buffer
			encoder.Reset(&buffer)
			if _, err := encoder.Write(data); err != nil {
				return err
			}
			if err := encoder.Close(); err != nil {
				return err
			}
			if buffer.Len() < len(data) {
				item.compressed = buffer.Bytes()
				item.compressedETag = etag(item.compressed)
			}
		}
		variants[name] = item
		return nil
	}); err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		item, exists := variants[name]
		if !exists {
			if name == "assets" || strings.HasPrefix(name, "assets/") || strings.Contains(path.Base(name), ".") {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
			item = variants[name]
			r2 := r.Clone(r.Context())
			u := *r.URL
			u.Path = "/"
			r2.URL = &u
			r = r2
		}
		if strings.HasPrefix(name, "assets/") && hashedAsset.MatchString(path.Base(name)) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		w.Header().Set("ETag", item.etag)
		if len(item.compressed) > 0 {
			w.Header().Add("Vary", "Accept-Encoding")
			// Keep byte ranges in the original file's coordinate space.
			if r.Header.Get("Range") == "" && r.URL.Path != "/index.html" && transfer.AcceptsGzip(r) {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Length", strconv.Itoa(len(item.compressed)))
				w.Header().Set("Content-Type", item.mediaType)
				w.Header().Set("ETag", item.compressedETag)
				http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(item.compressed))
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}

func etag(data []byte) string {
	return fmt.Sprintf(`"%x"`, sha256.Sum256(data))
}

func compressible(name string) bool {
	switch path.Ext(name) {
	case ".html", ".js", ".mjs", ".css", ".json", ".svg", ".txt", ".xml", ".webmanifest":
		return true
	default:
		return false
	}
}
