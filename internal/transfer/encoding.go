package transfer

import (
	"net/http"
	"strconv"
	"strings"
)

// AcceptsGzip reports whether gzip is acceptable and is not explicitly ranked below identity.
func AcceptsGzip(r *http.Request) bool {
	gzipQuality, wildcardQuality, identityQuality := -1.0, -1.0, 0.0
	for _, line := range r.Header.Values("Accept-Encoding") {
		for _, value := range strings.Split(line, ",") {
			coding, parameters, _ := strings.Cut(value, ";")
			quality := 1.0
			for _, parameter := range strings.Split(parameters, ";") {
				name, raw, found := strings.Cut(parameter, "=")
				if found && strings.EqualFold(strings.TrimSpace(name), "q") {
					q, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
					if err != nil || !(q >= 0 && q <= 1) {
						q = 0
					}
					quality = q
					break
				}
			}
			switch strings.ToLower(strings.TrimSpace(coding)) {
			case "gzip":
				gzipQuality = quality
			case "*":
				wildcardQuality = quality
			case "identity":
				identityQuality = quality
			}
		}
	}
	if gzipQuality < 0 {
		gzipQuality = wildcardQuality
	}
	return gzipQuality > 0 && gzipQuality >= identityQuality
}
