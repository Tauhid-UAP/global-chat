package middleware

import (
	"crypto/subtle"
	"net/http"
)

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			formToken := r.FormValue("CSRFToken")
			sessionToken := r.Context().Value(CSRFKey).(string)

			if subtle.ConstantTimeCompare(
				[]byte(formToken),
				[]byte(sessionToken),
			) != 1 {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

