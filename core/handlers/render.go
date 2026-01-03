package handlers

import (
	"log"
	"embed"
	"html/template"
	"net/http"


	"github.com/Tauhid-UAP/global-chat/core/models"
)

//The below comment is required so that the compiler embeds the HTML file in the build
//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type PageData struct {
	Title string
	User models.User
	CSRF string
	StaticAssetBaseURL template.URL
}

func Render(w http.ResponseWriter, page string, data PageData) {
	template, err := template.ParseFS(
		templateFS,
		"templates/base.html",
		"templates/"+page,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	log.Printf("Static base before execute: %s", data.StaticAssetBaseURL)
	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
