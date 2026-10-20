// Package middleware groups the cross-cutting HTTP middlewares: logging,
// panic recovery, and CORS (Technical Plan section 9).
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging records each request's method, path, status, and duration.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// Recover turns a panic inside the handler into a 500 response, instead of
// crashing the process.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recuperado en %s %s: %v", r.Method, r.URL.Path, rec)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"error interno"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS allows access from any origin. This is a self-hosted, single-user
// app (RNF-01): simplicity is prioritized so a future client/frontend can
// consume the API without extra configuration.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// APIKey requires the X-API-Key header when expectedKey isn't empty
// (RNF-01: optional simple authentication when exposing the backend outside
// localhost). With an empty expectedKey, the middleware does nothing.
func APIKey(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if expectedKey == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != expectedKey {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"no autorizado"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(status int) {
	sw.status = status
	sw.ResponseWriter.WriteHeader(status)
}
