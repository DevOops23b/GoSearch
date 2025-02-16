package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	//Tilføjet disse pakker grundet search funktion
	"html/template" // til html-sider(skabeloner
	"net/http" // til http-servere og håndtering af routere

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	//"github.com/mattn/go-sqlite3"
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

func main() {
	initDB()
	defer closeDB()
}


//////////////////////////////////////////////////////////////////////////////////
/// Page Routes & API Routes
//////////////////////////////////////////////////////////////////////////////////

// Detter er Gorilla Mux's route handler, i stedet for Flasks indbyggede router-handler
///Opretter en ny router
r := mux.NewRouter() 
//Definerer routerne.
r.HandleFunc("/", searchHandler) // hjemmeside ruten
r.HandleFunc("/api/search", apiSearchHandler) // API-ruten for søgninger.





//////////////////////////////////////////////////////////////////////////////////
/// Search Funktions
//////////////////////////////////////////////////////////////////////////////////


func searchHandler (w http.ResponseWritter, r *http.Request) {
	//Henter search-query fra URL-parameteren.
	query := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var searchResults []map[string]interface{
		if query!= "" {
			rows, err != queryDB(
				"SELECT * FROM pages WHERE language = ? 
				AND content LIKE ?", language, "%"+query+"%")
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			for rowsNext() {
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

		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, map[string]interface{}{
			"search_results": searchResults,
			"query": 		  query,
		})
	}
}


//////////////////////////////////////////////////////////////////////////////////
/// Security Functions
//////////////////////////////////////////////////////////////////////////////////




//////////////////////////////////////////////////////////////////////////////////
/// Main
//////////////////////////////////////////////////////////////////////////////////

func main() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatalf("Errorconnecting to database: %v", err)
	}
	defer closeDB()
}

