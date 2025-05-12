package main

import (
	"html/template"
	"log"
	"net/http"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"]

	data := map[string]any{
		"Title":        "Home",
		"UserLoggedIn": ok && userID != nil,
	}

	tmpl, err := template.ParseFiles(templatePath+"layout.html", templatePath+"index.html")
	if err != nil {
		log.Printf("Error parsing templates: %v", err)
		http.Error(w, "Error loading templates", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"]

	data := map[string]interface{}{
		"UserLoggedIn": ok && userID != nil,
	}

	tmpl, err := template.ParseFiles(templatePath+"layout.html", templatePath+"about.html")
	if err != nil {
		http.Error(w, "Error loading about-side", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
