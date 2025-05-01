package main

import (
	"html/template"
	"log"
	"net/http"
)

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "din valgte by"
	}

	data := struct {
		City    string
		Message string
	}{
		City:    city,
		Message: "Solen skinner i " + city + "!",
	}

	tmpl, err := template.ParseFiles(templatePath + "weather.html")
	if err != nil {
		http.Error(w, "Error loading weather page", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
