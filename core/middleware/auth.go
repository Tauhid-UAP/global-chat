package middleware

import (
	"context"
	"net/http"
	"strings"
	"github.com/Tauhid-UAP/global-chat/core/auth"

	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

type ctxKey string

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	CSRFKey contextKey = "csrf"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		key := "session:" + c.Value

		data, err := redisclient.Client.HGetAll(r.Context(), key).Result()

		if (err != nil) || (len(data) == 0) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, data["user_id"])
		ctx = context.WithValue(ctx, CSRFKey, data["csrf_token"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}



func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			parts := strings.Split(cookie.Value, "|")
			if len(parts) != 2 {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userID := parts[0]
			signature := parts[1]

			if !auth.Verify(userID, signature, secret) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
