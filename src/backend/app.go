package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	//Tilføjet disse pakker grundet search funktion
	"encoding/json" // Gør at vi kan læse json-format
	"html/template" // til html-sider(skabeloner)
	"net/http" // til http-servere og håndtering af routere

	// en router til http-requests
	"github.com/gorilla/mux"

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	_ "github.com/mattn/go-sqlite3"
)

//////////////////////////////////////////////////////////////////////////////////
/// Database Functions
//////////////////////////////////////////////////////////////////////////////////

const (
	DATABASE_PATH = "../whoknows.db"
)

var db *sql.DB

//////////////////////////////////////////////////////////////////////////////////
/// Database Functions
//////////////////////////////////////////////////////////////////////////////////

func checkDBExists() bool {
	if _, err := os.Stat(DATABASE_PATH); os.IsNotExist(err) {
		fmt.Println("Database not found")
		return false
	}
	return true
}

func connectDB() (*sql.DB, error) {
	if !checkDBExists() {
		log.Fatal("Database does not exist")
	}

	db, err := sql.Open("sqlite3", DATABASE_PATH)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	return db, nil

}

func initDB() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func queryDB(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func executeDB(query string, args ...interface{}) (sql.Result, error) {
	return db.Exec(query, args...)
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}


//////////////////////////////////////////////////////////////////////////////////
/// Root handlers
//////////////////////////////////////////////////////////////////////////////////

// Viser forside
func rootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

//////////////////////////////////////////////////////////////////////////////////
/// Search handler
//////////////////////////////////////////////////////////////////////////////////

// Viser search api-server.
/*func searchHandler (w http.ResponseWriter, r *http.Request) {
	//Henter search-query fra URL-parameteren.
	query := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	//Henter query-parameterne
	var searchResults []map[string]interface{}
	if query!= "" {
		rows, err := queryDB(
			"SELECT * FROM pages WHERE language = ? AND content LIKE ?", 
			language, "%"+query+"%")
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
			
		// SQL-forespørgsel - finder sider i databasen, hvor 'content' matcher 'query'
		for rows.Next() {
			var title, url, description string
			if err := rows.Scan(&title, &url, &description); err != nil {
				http.Error(w, "Error reading row", http.StatusInternalServerError)
				return
			}
			searchResults = append(searchResults, map[string]interface{}{
				"title":	title,
				"url":		url,
				"description": description,
			})
		}


	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"search_results": searchResults,
	})
	
}*/

// Med dummydata for at teste endpointsne
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Hent query-parametre
	query := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	// Dummy data til test
	searchResults := []map[string]interface{}{
		{"title": "Mock Page 1", "url": "http://test1.com", "description": "This is a test result 1"},
		{"title": "Mock Page 2", "url": "http://test2.com", "description": "This is a test result 2"},
	}

	// Log query til terminalen (for debugging)
	fmt.Printf("Search query: %s, Language: %s\n", query, language)

	// Sæt Content-Type til JSON og send svar
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"search_results": searchResults,
	})
}




//////////////////////////////////////////////////////////////////////////////////
/// Security Functions
//////////////////////////////////////////////////////////////////////////////////




//////////////////////////////////////////////////////////////////////////////////
/// Main
//////////////////////////////////////////////////////////////////////////////////

func main() {
	// initialiserer databasen og forbinder til den. 
	//initDB() // skal udkommenteres under test af search dummy-data
	//defer closeDB() // skal udkommenteres under test af search dummy-data

	// Detter er Gorilla Mux's route handler, i stedet for Flasks indbyggede router-handler
	///Opretter en ny router
	r := mux.NewRouter() 
	//Definerer routerne.
	r.HandleFunc("/", rootHandler).Methods("GET") // Forside
	r.HandleFunc("/api/search", searchHandler).Methods("GET") // API-ruten for søgninger.


	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

}

