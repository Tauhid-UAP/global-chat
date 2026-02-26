package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"log"
	"time"

	"github.com/Tauhid-UAP/global-chat/core/middleware"
	"github.com/Tauhid-UAP/global-chat/core/store"

	"github.com/Tauhid-UAP/global-chat/core/awsclient"
)

func Profile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	user, _ := store.GetUserByID(r.Context(), userID)

	if r.Method == http.MethodPost {
		user.FirstName = r.FormValue("FirstName")
		user.LastName = r.FormValue("LastName")

		file, header, err := r.FormFile("ProfileImage")
		if err != nil {
			log.Printf("File upload error: %v", err)
			store.UpdateUser(r.Context(), user)
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}

		defer file.Close()
		
		path, err := awsclient.Service.UploadFile(r.Context(), fmt.Sprintf("golang-sample-web-profile-images/%s_%d%s", userID, time.Now().Unix(), filepath.Ext(header.Filename)), file, header.Header.Get("Content-Type"))
		if err != nil {
			log.Fatal(err)
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}

		/*
		path := filepath.Join("media/uploads", userID+"_"+header.Filename)
		destination, _ := os.Create(path)
		defer destination.Close()

		io.Copy(destination, file)
		*/

		user.ProfileImage = &path

		store.UpdateUser(r.Context(), user)
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	Render(w, "profile.html", PageData{
		Title: "Profile",
		User: user,
		CSRF: r.Context().Value(middleware.CSRFKey).(string),
	})
}
