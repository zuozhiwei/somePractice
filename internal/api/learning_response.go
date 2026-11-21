package api

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"skilltree-go/internal/transfer"
)

var learningGzipWriters = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// okLearning opts course reads into compression. Other replies, including tokens and
// recovery codes, use ok or reply and are outside this path.
func okLearning(w http.ResponseWriter, r *http.Request, data any) {
	writeLearning(w, r, 0, data)
}

// The validators are for the data held by the open page, including its personal progress.
func okLearningWithETag(w http.ResponseWriter, r *http.Request, user int64, data any) {
	writeLearning(w, r, user, data)
}

func writeLearning(w http.ResponseWriter, r *http.Request, user int64, data any) {
	w.Header().Add("Vary", "Accept-Encoding")
	useGzip := transfer.AcceptsGzip(r)
	if user == 0 && !useGzip {
		ok(w, data)
		return
	}

	body, err := json.Marshal(responseEnvelope{Code: http.StatusOK, Data: data, Message: "success"})
	if err != nil {
		fail(w)
		return
	}
	body = append(body, '\n')
	if user != 0 {
		digest := sha256.New()
		_, _ = io.WriteString(digest, strconv.FormatInt(user, 10)+"\n")
		_, _ = digest.Write(body)
		// Identity and gzip encode the same JSON. Give both forms a weak validator.
		tag := fmt.Sprintf(`W/"%x"`, digest.Sum(nil))
		w.Header().Set("ETag", tag)
		if matchesLearningETag(r, tag) {
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	if useGzip && len(body) >= 1024 {
		var compressed bytes.Buffer
		writer := learningGzipWriters.Get().(*gzip.Writer)
		writer.Reset(&compressed)
		_, _ = writer.Write(body)
		_ = writer.Close()
		writer.Reset(io.Discard)
		learningGzipWriters.Put(writer)
		if compressed.Len() < len(body) {
			w.Header().Set("Content-Encoding", "gzip")
			body = compressed.Bytes()
		}
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writeJSONStatus(w, http.StatusOK)
	_, _ = w.Write(body)
}

func matchesLearningETag(r *http.Request, tag string) bool {
	want := strings.TrimPrefix(tag, "W/")
	for _, header := range r.Header.Values("If-None-Match") {
		for _, value := range strings.Split(header, ",") {
			value = strings.TrimSpace(value)
			if value == "*" || strings.TrimPrefix(value, "W/") == want {
				return true
			}
		}
	}
	return false
}
