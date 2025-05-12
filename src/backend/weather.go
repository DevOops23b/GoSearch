package main

import (
	"html/template"
	"log"
	"net/http"
)

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"]

	city := r.URL.Query().Get("city")
	if city == "" {
		city = "din valgte by"
	}

	data := struct {
		Title        string
		City         string
		Message      string
		UserLoggedIn bool
		Template     string
	}{
		Title:        "Weather",
		City:         city,
		Message:      "Solen skinner i " + city + "!",
		UserLoggedIn: ok && userID != nil,
		Template:     "weather.html",
	}

	tmpl, err := template.ParseFiles(templatePath+"layout.html", templatePath+"weather.html")
	if err != nil {
		http.Error(w, "Error loading weather page", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
