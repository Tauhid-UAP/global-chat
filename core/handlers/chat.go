package handlers

import (
	"log"
	"net/http"
	"html/template"
	"github.com/Tauhid-UAP/golang-sample-web-app/core/middleware"
)

func ChatPageHandler(staticAssetBaseURL template.URL) http.HandlerFunc {
	log.Printf("Static base injected: %s", staticAssetBaseURL)
	return func (w http.ResponseWriter, r *http.Request) {
		log.Printf("Static base before render: %s", staticAssetBaseURL)
		Render(w, "chat.html", PageData{
			Title: "Global Chat",
			CSRF: r.Context().Value(middleware.CSRFKey).(string),
			StaticAssetBaseURL: staticAssetBaseURL,
		})
	}
}
