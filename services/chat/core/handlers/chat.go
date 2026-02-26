package handlers

import (
	"log"
	"net/http"
	"html/template"

	"github.com/Tauhid-UAP/global-chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/core/userselector"
)

func ChatPageHandler(staticAssetBaseURL template.URL) http.HandlerFunc {
	log.Printf("Static base injected: %s", staticAssetBaseURL)
	return func (w http.ResponseWriter, r *http.Request) {
		log.Printf("Static base before render: %s", staticAssetBaseURL)

		userID := r.Context().Value(middleware.UserIDKey).(string)
		isAnonymousUser := r.Context().Value(middleware.IsAnonymousUserKey).(bool)
		
		user, _ := userselector.GetUserByIDFromApplicableStore(r.Context(), userID, isAnonymousUser)

		Render(w, "chat.html", PageData{
			Title: "Global Chat",
			User: user,
			CSRF: r.Context().Value(middleware.CSRFKey).(string),
			StaticAssetBaseURL: staticAssetBaseURL,
		})
	}
}
