package handlers

import (
	"net/http"
	"log"
	"time"
	"html/template"

	"github.com/google/uuid"
	"github.com/Tauhid-UAP/global-chat/core/auth"
	"github.com/Tauhid-UAP/global-chat/core/models"
	"github.com/Tauhid-UAP/global-chat/core/store"

	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

func RegisterHandler(staticAssetBaseURL template.URL) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			Render(w, "register.html", PageData{
				Title: "Register",
				StaticAssetBaseURL: staticAssetBaseURL,
			})
			return
		}

		email := r.FormValue("Email")
		password := r.FormValue("Password")
		firstName := r.FormValue("FirstName")
		lastName := r.FormValue("LastName")

		passwordHash, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error", 500)
			return
		}

		user := models.User{
			ID: uuid.NewString(),
			Email: email,
			FirstName: firstName,
			LastName: lastName,
			PasswordHash: passwordHash,
		}

		err = store.CreateUser(r.Context(), user)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, "Registration failed", 400)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LoginHandler(staticAssetBaseURL template.URL) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			Render(w, "login.html", PageData{
				Title: "Login",
				StaticAssetBaseURL: staticAssetBaseURL,
			})
			return
		}

		email := r.FormValue("Email")
		password := r.FormValue("Password")

		user, err := store.GetUserByEmail(r.Context(), email)
		if (err != nil) || !auth.VerifyPassword(user.PasswordHash, password) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		sessionID, _, err := auth.CreateSession(r.Context(), user.ID, 24*time.Hour)
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

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_id")
	if err == nil {
		redisclient.Client.Del(r.Context(), "session:"+c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session_id",
		Value: "",
		Path: "",
		MaxAge: -1, // expire immediately
		HttpOnly: true,
		Secure: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
