package server

import (
	"net/http"

	"github.com/flant/negentropy/authd/pkg/log"
)

func DebugAwareLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		debug := r.URL.Query().Get("debug")
		if debug == "yes" {
			r = r.WithContext(log.WithDebugLogger(r.Context()))
		} else {
			r = r.WithContext(log.WithStandardLogger(r.Context()))
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
