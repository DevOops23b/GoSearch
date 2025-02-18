package main
import (
	"database/sql"
	"fmt"
	"log"
	"os"
	//"golang.org/x/crypto/bcrypt" Will be added later
	//Tilføjet disse pakker grundet search funktion
	//"encoding/json" // Gør at vi kan læse json-format
	"html/template" // til html-sider(skabeloner)
	"net/http" // til http-servere og håndtering af routere

	// en router til http-requests
	"github.com/gorilla/mux"

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	_ "github.com/mattn/go-sqlite3"


	"github.com/gorilla/sessions"
)

//////////////////////////////////////////////////////////////////////////////////
/// Database Functions
//////////////////////////////////////////////////////////////////////////////////

const (
	DATABASE_PATH = "../whoknows.db"
)

var db *sql.DB

var store = sessions.NewCookieStore([]byte("Very-secret-key"))

type User struct {
	ID			int		`json:"id"`
	Username	string	`json:"username"`
	Password	string	`json:"password"`
}

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
	tmpl, err := template.ParseFiles("../frontend/templates/index.html")
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
	var searchResults []map[string]string
	if query != "" {
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
			searchResults = append(searchResults, map[string]string{
				"title":	title,
				"url":		url,
				"description": description,
			})
		}

	}

	//indlæser search.htmlmed resultater
	tmpl, err := template.ParseFiles("../frontend/templates/search.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	// sender data til html-templaten
	tmpl.Execute(w, map[string]interface{}{
		"Query": 	query,
		"Results": 	searchResults,
	})


	//Sendte json objekter
	/*w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"search_results": searchResults,
	})
	
}*/

// Med dummydata for at teste endpointsne
func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	// Dummy data til test
	searchResults := []map[string]string{
		{"Title": "Mock Page 1", "URL": "http://test1.com", "Description": "This is a test result 1"},
		{"Title": "Mock Page 2", "URL": "http://test2.com", "Description": "This is a test result 2"},
	}

	// Log query til terminalen (for debugging)
	fmt.Printf("Search query: %s, Language: %s\n", query, language)

	// Indlæs search.html med dummyresultater
	tmpl, err := template.ParseFiles("../frontend/templates/search.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	// Send data til HTML-templaten
	tmpl.Execute(w, map[string]interface{}{
		"Query":   query,
		"Results": searchResults,
	})
}


//////////////////////////////////////////////////////////////////////////////////
/// Page routes
//////////////////////////////////////////////////////////////////////////////////

func login(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name") //Due to err, the error will not be ignored
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["user_id"]; ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	tmpl := template.Must(template.ParseFiles("../frontend/templates/login.html"))
	tmpl.Execute(w, nil)

}




//////////////////////////////////////////////////////////////////////////////////
/// API routes
//////////////////////////////////////////////////////////////////////////////////

func apiLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid request!!!", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user User

	err = db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Password)

	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if user.Password != password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)


}



//////////////////////////////////////////////////////////////////////////////////
/// Security Functions
//////////////////////////////////////////////////////////////////////////////////




//////////////////////////////////////////////////////////////////////////////////
/// Main
//////////////////////////////////////////////////////////////////////////////////

func requestHandler() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/login", login).Methods("GET")
	router.HandleFunc("/api/login", apiLogin).Methods("POST")
}


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
	// sørger for at vi kan bruge de statiske filer som ligger i static-mappen. ex: css.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("../frontend/static/"))))
	requestHandler()
	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

	

}
