package middleware

import (
	"context"
	"net/http"
	"strings"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Tauhid-UAP/global-chat/core/auth"
	"github.com/Tauhid-UAP/global-chat/core/models"
	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	IsAnonymousUserKey contextKey = "is_anonymous_user"
	CSRFKey contextKey = "csrf_token"
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

		isAnonymousUser, isAnonymousUserParseError := strconv.ParseBool(data["is_anonymous_user"])
		if (isAnonymousUser) || (isAnonymousUserParseError != nil) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, data["user_id"])
		ctx = context.WithValue(ctx, IsAnonymousUserKey, false)
		ctx = context.WithValue(ctx, CSRFKey, data["csrf_token"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			anonymousUserID := uuid.NewString()
			isAnonymousUser := true
			anonymousUser := models.User {
				ID: anonymousUserID,
				FirstName: "Anonymous",
				LastName: anonymousUserID[len(anonymousUserID) - 6: ],
				IsAnonymous: isAnonymousUser,
			}

			currentContext := r.Context()
			anonymousUserKey := "anonymous_user:" + anonymousUserID
			err := redisclient.AddUser(currentContext, anonymousUserKey, anonymousUserID, anonymousUser.FirstName, anonymousUser.LastName, isAnonymousUser)

			if err != nil {
				return
			}

			redisExpireErr := redisclient.Client.Expire(currentContext, anonymousUserKey, 2*time.Hour).Err()

			if redisExpireErr != nil {
				return
			}

			sessionID, CSRFToken, err := auth.CreateSession(r.Context(), anonymousUserID, isAnonymousUser, 24*time.Hour)
			if err != nil {
				http.Error(w, "Session error", 500)
				return
			}
		
			http.SetCookie(w, &http.Cookie{
				Name: "session_id",
				Value: sessionID,
				Path: "/",
				HttpOnly: true,
				Secure: true,
				SameSite: http.SameSiteLaxMode,
			})
			
			ctx := context.WithValue(currentContext, UserIDKey, anonymousUserID)
			ctx = context.WithValue(ctx, IsAnonymousUserKey, true)
			ctx = context.WithValue(ctx, CSRFKey, CSRFToken)

			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		key := "session:" + c.Value

		data, err := redisclient.Client.HGetAll(r.Context(), key).Result()

		if (err != nil) || (len(data) == 0) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

                isAnonymousUser, isAnonymousUserParseError := strconv.ParseBool(data["is_anonymous_user"])
                if isAnonymousUserParseError != nil {
                        http.Redirect(w, r, "/login", http.StatusSeeOther)
                        return
                }

		ctx := context.WithValue(r.Context(), UserIDKey, data["user_id"])
		ctx = context.WithValue(ctx, IsAnonymousUserKey, isAnonymousUser)
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
