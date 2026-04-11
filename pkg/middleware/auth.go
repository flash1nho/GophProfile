package middleware

import (
	"net/http"
)

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-ID") == "" {
			http.Error(w, "missing X-User-ID", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
