// Package middleware agrupa los middlewares HTTP transversales: logging,
// recuperación de panics y CORS (Plan Técnico sección 9).
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging registra método, path, status y duración de cada request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// Recover convierte un panic en un dentro del handler en una respuesta 500,
// en vez de tumbar el proceso.
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

// CORS habilita acceso desde cualquier origen. Es una app single-user
// self-hosted (RNF-01): se prioriza simplicidad para poder consumir la API
// desde un futuro cliente/frontend sin configuración adicional.
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

// APIKey exige el header X-API-Key cuando expectedKey no está vacío
// (RNF-01: autenticación simple opcional al exponer el backend fuera de
// localhost). Con expectedKey vacío el middleware no hace nada.
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
