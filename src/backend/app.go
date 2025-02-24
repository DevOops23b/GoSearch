package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"strings"

	"encoding/json" // Gør at vi kan læse json-format
	"html/template" // til html-sider(skabeloner)
	"net/http"      // til http-servere og håndtering af routere

	// en router til http-requests
	"github.com/gorilla/mux"

	//Til at hashe passwords
	"golang.org/x/crypto/bcrypt"

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/gorilla/sessions"
)

//////////////////////////////////////////////////////////////////////////////////
/// Configurations
//////////////////////////////////////////////////////////////////////////////////

const (
	DATABASE_PATH = "../whoknows.db"
)

var db *sql.DB

var store = sessions.NewCookieStore([]byte("Very-secret-key"))

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PageData struct {
	User         *User
	Error        string
	Title        string
	Template     string
	UserLoggedIn bool
}

type Page struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Language    string    `json:"language"`
	LastUpdated time.Time `json:"last_updated"`
	Content     string    `json:"content"`
}

type WeatherResponse struct {
	Name 		string    `jason:"name"`
	Main 		struct {
		Temp float64 	  `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`

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
func checkTables() {
	// Check users table
	fmt.Println("\n--- Users in database ---")
	rows, err := queryDB("SELECT * FROM users")
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
		if err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		fmt.Printf("ID: %d, Username: %s, Email: %s\n", user.ID, user.Username, user.Email)
	}

	// Check pages table
	fmt.Println("\n--- Pages in database ---")
	rows2, err := queryDB("SELECT * FROM pages")
	if err != nil {
		log.Printf("Error querying pages: %v", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var page Page
		err := rows2.Scan(&page.Title, &page.URL, &page.Language, &page.LastUpdated, &page.Content)
		if err != nil {
			log.Printf("Error scanning page: %v", err)
			continue
		}
		fmt.Printf("Title: %s, URL: %s, Language: %s\n", page.Title, page.URL, page.Language)
	}
}

//////////////////////////////////////////////////////////////////////////////////
/// Root handlers
//////////////////////////////////////////////////////////////////////////////////

// Viser forside
func rootHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"]

	data := map[string]any{
		"Title":        "Home",
		"UserLoggedIn": ok && userID != nil,
	}

	tmpl, err := template.ParseFiles("../frontend/templates/index.html")
	if err != nil {
		http.Error(w, "Error loading index-side", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

//////////////////////////////////////////////////////////////////////////////////
/// about handler
//////////////////////////////////////////////////////////////////////////////////

// Viser about-siden
func aboutHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("../frontend/templates/about.html")
	if err != nil {
		http.Error(w, "Error loading about-side", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

//////////////////////////////////////////////////////////////////////////////////
/// Search handler
//////////////////////////////////////////////////////////////////////////////////

// Viser search api-server.
func searchHandler(w http.ResponseWriter, r *http.Request) {
	//Henter search-query fra URL-parameteren.
    query := strings.TrimSpace(r.URL.Query().Get("q"))
	language := r.URL.Query().Get("language")

	if language == "" {
		language = "en"
	}

	//Henter query-parameterne
	var searchResults []map[string]string
	if query != "" {

		// Viser hvad der bliver sendt i SQL-forespørgelsen
		fmt.Printf("Query: %s, Language: %s\n", query, language)

		rows, err := queryDB(
			"SELECT title, url, content FROM pages WHERE language = ? AND LOWER(title) LIKE LOWER(?)",
			language, "%"+query+"%",
		)

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		defer rows.Close()

		// SQL-forespørgsel - finder sider i databasen, hvor 'content' matcher 'query'
		for rows.Next() {
			var title, url, content string
			if err := rows.Scan(&title, &url, &content); err != nil {
				http.Error(w, "Error reading row", http.StatusInternalServerError)
				return
			}
			searchResults = append(searchResults, map[string]string{
				"title":       title,
				"url":         url,
				"description": content,
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
		"Query":   query,
		"Results": searchResults,
	})
}

//////////////////////////////////////////////////////////////////////////////////
/// Login/Logout Handlers ???
//////////////////////////////////////////////////////////////////////////////////

var tmpl = template.Must(template.ParseFiles("../frontend/templates/layout.html", "../frontend/templates/login.html"))

func login(w http.ResponseWriter, r *http.Request) {

	data := PageData{
		Title:    "Log in",
		Template: "login",
	}

	err := tmpl.ExecuteTemplate(w, "layout.html", data)

	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)

	}

	/*
		data := map[string]interface{} {
			"Error": "", // default error message
		}

		session, err := store.Get(r, "session-name") //Due to err, the error will not be ignored
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if _, ok := session.Values["user_id"]; ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}




		tmpl.Execute(w, nil)
		tmpl.ExecuteTemplate(w, "layout.html", data)
	*/

}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["user_id"]; !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session.Options.MaxAge = -1 // Expire the session
	delete(session.Values, "user_id")

	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//////////////////////////////////////////////////////////////////////////////////
/// Login Handlers
//////////////////////////////////////////////////////////////////////////////////

func apiLogin(w http.ResponseWriter, r *http.Request) {
	/*
		data := map[string]interface{} {
			"Error": "", // default error message
		}

	*/

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
		data := PageData{Error: "Invalid username"}
		tmpl.ExecuteTemplate(w, "login.html", data)
		//http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	validated := validatePassword(user.Password, password)

	if validated == false {
		data := PageData{Error: "Invalid password"}
		tmpl.ExecuteTemplate(w, "login.html", data)
		return
	}

	if username == "" || password == "" {
		data := PageData{Error: "Please enter both username and password"}
		tmpl.ExecuteTemplate(w, "login.html", data)
	}

	if user.Password != password {
		data := PageData{Error: "Invalid password"}
		w.WriteHeader(http.StatusUnauthorized)
		tmpl.ExecuteTemplate(w, "login.html", data)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)

}


//////////////////////////////////////////////////////////////////////////////////
/// Register handlers
//////////////////////////////////////////////////////////////////////////////////

// viser registreringssiden.
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if userIsLoggedIn(r) {
		http.Redirect(w, r, "/search", http.StatusFound)
		return
	}
	tmpl, err := template.ParseFiles("../frontend/templates/register.html")
	if err != nil {
		http.Error(w, "Error loading register page", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

//Håndterer registreringsprocessen
func apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if userIsLoggedIn(r) {
		http.Redirect(w, r, "/search", http.StatusFound)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	if username == "" {
		http.Error(w, "You have to enter a username", http.StatusBadRequest)
		return
	}
	if email == "" || !isValidEmail(email) {
		http.Error(w, "You have to enter a valid email address", http.StatusBadRequest)
		return
	}
	if password == "" {
		http.Error(w, "You have to enter a password", http.StatusBadRequest)
		return
	}
	if password != password2 {
		http.Error(w, "The two passwords do not match", http.StatusBadRequest)
		return
	}

	// Tjek for eksisterende brugernavn og email
	usernameTaken, emailTaken := userExists(username, email)
	if usernameTaken {
		http.Error(w, "The username is already taken", http.StatusBadRequest)
		return
	}

	if emailTaken {
		http.Error(w, "A user with this email already exists", http.StatusBadRequest)
		return
	}

	// Hash passwordet
	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Indsæt brugeren i databasen
	_, err = db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, hashedPassword)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	} 

	// Redirect til login-siden
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}


//Ser om brugere allerede eksisterer
func userExists(username, email string) (bool, bool) {
	var usernameExists, emailExists bool
	
	// Tjekker for eksisterende brugernavn
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=?)", username).Scan(&usernameExists)
	if err != nil && err != sql.ErrNoRows {
		return false, false // Fejl i forespørgslen
	}

	// Tjekker for eksisterende email
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email=?)", email).Scan(&emailExists)
	if err != nil && err != sql.ErrNoRows {
		return false, false // Fejl i forespørgslen
	}

	return usernameExists, emailExists
}

// tjekker om brugeren er logget in

func userIsLoggedIn(r *http.Request) bool {
	// skal have mere, når login-er implementeret.
	return false
}


// simpel email validering indtil videre kun med .com. skal udvides. 
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email) // Fjern mellemrum
	if !strings.Contains(email, "@") {
		return false
	}
		
	// Tjek at den ikke starter eller slutter med "@"
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	validEndings := []string{".com", ".dk", ".net", ".org", ".edu"}

	for _, ending := range validEndings {
		if strings.HasSuffix(email, ending) {
			return true
		}
	}
	return false
}

//////////////////////////////////////////////////////////////////////////////////
/// Weather handler
//////////////////////////////////////////////////////////////////////////////////

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("WEATHER_API_KEY") //SKal sættes i vores miljø-variabler.
	city := r.URL.Query().Get("city") //henter city, fra query parameter
	if city == "" {
		http.Error(w, "City name is required", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, apiKey)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var weather WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		http.Error(w, "Error decoding weather data", http.StatusInternalServerError)
		return
	}

	data := struct {
		Name     string
        Temp     float64
        Weather  string
    }{
        Name:    weather.Name,
        Temp:    weather.Main.Temp,
        Weather: weather.Weather[0].Description,
	}

	tmpl, err := template.ParseFiles("../frontend/templates/weather.html")
    if err != nil {
        http.Error(w, "Error loading weather template", http.StatusInternalServerError)
        return
    }

    // Render skabelonen med vejrinformationen
    if err := tmpl.Execute(w, data); err != nil {
        http.Error(w, "Error rendering weather template", http.StatusInternalServerError)
    }

}


//////////////////////////////////////////////////////////////////////////////////
/// Security Functions
//////////////////////////////////////////////////////////////////////////////////

//hasher passwordet med bcrypt.
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func validatePassword(hashedPassword, inputPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	return err == nil
}

//////////////////////////////////////////////////////////////////////////////////
/// Main
//////////////////////////////////////////////////////////////////////////////////

func main() {
	// initialiserer databasen og forbinder til den.
	initDB()
	defer closeDB()

	checkTables()

	err := db.Ping()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	fmt.Println("Database connection successful!")

	// Detter er Gorilla Mux's route handler, i stedet for Flasks indbyggede router-handler
	///Opretter en ny router
	r := mux.NewRouter()

	//Definerer routerne.
	r.HandleFunc("/", rootHandler).Methods("GET")       // Forside
	r.HandleFunc("/about", aboutHandler).Methods("GET") //about-side
	r.HandleFunc("/login", login).Methods("GET")		//Login-side
	r.HandleFunc("/register", registerHandler).Methods("GET") //Register-side

	// Definerer api-erne
	r.HandleFunc("/api/login", apiLogin).Methods("POST")
	r.HandleFunc("/api/search", searchHandler).Methods("GET") 
	r.HandleFunc("/api/logout", logoutHandler).Methods("GET") 
	r.HandleFunc("/api/search", searchHandler).Methods("GET") // API-ruten for søgninger.
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	r.HandleFunc("/api/weather", weatherHandler).Methods("GET") //weather-side


	// sørger for at vi kan bruge de statiske filer som ligger i static-mappen. ex: css.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("../frontend/static/"))))

	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

}
