package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	//Elasticsearch v8 client
	"github.com/elastic/go-elasticsearch/v8"

	//"encoding/json" // Gør at vi kan læse json-format
	"html/template" // til html-sider(skabeloner)
	"net/http"      // til http-servere og håndtering af routere

	// en router til http-requests
	"github.com/gorilla/mux"

	//Til at hashe passwords
	"golang.org/x/crypto/bcrypt"

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/gorilla/sessions"

	"github.com/robfig/cron/v3"
	// Import the cron library to schedule periodic tasks
	// PostgreSQL driver instead of SQLite.
	_ "github.com/lib/pq"
)

///////////////////////////////////////////////////////////////////////////////////
/// Configurations
///////////////////////////////////////////////////////////////////////////////////

var CONN_STR string

var templatePath string

var staticPath string

func init() {

	if err := godotenv.Load(".env.local"); err != nil {
		// If .env.local doesn't exist, try regular .env
		if err := godotenv.Load(); err != nil {
			log.Println("No .env files found. Using environment variables.")
		}
	}

	CONN_STR = os.Getenv("CONN_STR")
	if CONN_STR == "" {
		log.Println("Warning: CONN_STR not set, using default connection string")
		CONN_STR = "postgres://youruser:yourpassword@localhost:5432/whoknows?sslmode=disable"
	}

	templatePath = os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "../frontend/templates/"
	}

	staticPath = os.Getenv("STATIC_PATH")
	if staticPath == "" {
		staticPath = "../frontend/static/"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Println("Warning: SESSION_SECRET not set, using default (insecure) secret")
		sessionSecret = "Very-secret-key"
	}

	store = sessions.NewCookieStore([]byte(sessionSecret))

}

var db *sql.DB
var esClient *elasticsearch.Client

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
	Name string `json:"name"`
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

//////////////////////////////////////////////////////////////////////////////////
/// Database Functions
//////////////////////////////////////////////////////////////////////////////////

func connectDB() (*sql.DB, error) {
	var db *sql.DB
	var err error

	maxRetries := 10
	retryDelay := time.Second * 5

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", CONN_STR)
		if err != nil {
			log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}
		err = db.Ping()
		if err == nil {
			log.Println("Successfully connected to PostgresSQL!")
			return db, nil
		}
		log.Printf("Database ping failed (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("failed to connect to database after %d attempts", maxRetries)

}

func initDB() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("PostgresSQL ping failed: %v", err)
	}
	log.Println("Connected to PostgresSQL!")

}

func queryDB(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
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

// /////////////////////////////////////////////////////////////////////////////////
// Elasticsearch Functions & Integration
// /////////////////////////////////////////////////////////////////////////////////
func initElasticsearch() {
	var err error
	maxRetries := 10
	retryDelay := time.Second * 5

	esHost := os.Getenv("ES_HOST")
	if esHost == "" {
		esHost = "localhost"
	}

	esPassword := os.Getenv("ES_PASSWORD")
	if esPassword == "" {
		esPassword = "changeme"
	}

	esUsername := os.Getenv("ES_USERNAME")
	if esUsername == "" {
		esUsername = "elastic"
	}

	for i := 0; i < maxRetries; i++ {
		// Try both HTTPS and HTTP connections
		configs := []elasticsearch.Config{
			// Try HTTP first
			{
				Addresses: []string{fmt.Sprintf("http://%s:9200", esHost)},
				Username:  esUsername,
				Password:  esPassword,
			},
			// Try HTTPS as fallback
			{
				Addresses: []string{fmt.Sprintf("https://%s:9200", esHost)},
				Username:  esUsername,
				Password:  esPassword,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		}

		// Try each config until one works
		for _, config := range configs {
			esClient, err = elasticsearch.NewClient(config)
			if err != nil {
				log.Printf("Error creating Elasticsearch client with config %v: %s", config.Addresses, err)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			res, err := esClient.Info(esClient.Info.WithContext(ctx))
			cancel()

			if err == nil {
				defer res.Body.Close()
				log.Printf("Successfully connected to Elasticsearch via %s", config.Addresses[0])
				return
			}

			log.Printf("Error connecting to Elasticsearch via %s: %v", config.Addresses[0], err)
		}

		log.Printf("Could not connect to Elasticsearch (attempt %d/%d). Retrying in %v...",
			i+1, maxRetries, retryDelay)
		time.Sleep(retryDelay)
	}

	log.Fatalf("Failed to connect to Elasticsearch after %d attempts", maxRetries)
}

func searchPagesInEs(query string) ([]Page, error) {
	var pages []Page

	searchBody := strings.NewReader(fmt.Sprintf(`{
		"query": {
			"multi_match": {
				"query": "%s",
				"fields": ["title^3", "url^2", "content"]
			}
		}
	}`, query))

	res, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex("pages"),
		esClient.Search.WithBody(searchBody),
		esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return pages, err
	}
	defer res.Body.Close()

	var r struct {
		Hits struct {
			Hits []struct {
				Source Page `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return pages, err
	}

	for _, hit := range r.Hits.Hits {
		pages = append(pages, hit.Source)
	}

	return pages, nil
}

func syncPagesToElasticsearch() error {
	rows, err := db.Query("SELECT title, url, content FROM pages")
	if err != nil {
		return fmt.Errorf("error querying pages from DB: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var page Page
		if err := rows.Scan(&page.Title, &page.URL, &page.Content); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		doc, err := json.Marshal(page)
		if err != nil {
			log.Printf("Error marshaling page: %v", err)
			continue
		}

		// Index the document without specifying a document type
		_, err = esClient.Index(
			"pages",                            // Index name
			strings.NewReader(string(doc)),     // JSON document
			esClient.Index.WithRefresh("true"), // Refresh immediately
		)
		if err != nil {
			log.Printf("Error indexing page to ES: %v", err)
			continue
		}

		count++
	}
	log.Printf("Synced %d pages to Elasticsearch", count)
	return nil
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

	tmpl, err := template.ParseFiles(templatePath + "index.html")
	if err != nil {
		http.Error(w, "Error loading index-side", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

//////////////////////////////////////////////////////////////////////////////////
/// about handler
//////////////////////////////////////////////////////////////////////////////////

// Viser about-siden
func aboutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["user_id"]

	data := map[string]interface{}{
		"UserLoggedIn": ok && userID != nil,
	}

	tmpl, err := template.ParseFiles(templatePath + "about.html")
	if err != nil {
		http.Error(w, "Error loading about-side", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

//////////////////////////////////////////////////////////////////////////////////
/// Search handler
//////////////////////////////////////////////////////////////////////////////////

// Viser search api-server.
func searchHandler(w http.ResponseWriter, r *http.Request) {
	//Henter search-query fra URL-parameteren.
	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	if queryParam == "" {
		http.Error(w, "No search query provided", http.StatusBadRequest)
		return
	}

	//Nuild search against Elasticsearch
	pages, err := searchPagesInEs(queryParam)
	if err != nil {
		log.Printf("Error searching Elasticsearch: %v", err)
		http.Error(w, "Error during search", http.StatusInternalServerError)
		return
	}

	// Build search results from Elasticsearch response
	var searchResults []map[string]string
	for _, page := range pages {
		searchResults = append(searchResults, map[string]string{
			"title":       page.Title,
			"url":         page.URL,
			"description": page.Content,
		})
	}

	tmpl, err := template.ParseFiles(templatePath + "search.html")
	if err != nil {
		http.Error(w, "Error loading search template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Query":   queryParam,
		"Results": searchResults,
	}); err != nil {
		log.Printf("Error executing search template: %v", err)
		http.Error(w, "Error rengering search results", http.StatusInternalServerError)
	}
}

//////////////////////////////////////////////////////////////////////////////////
/// Login/Logout Handlers ???
//////////////////////////////////////////////////////////////////////////////////

func getTemplates() (*template.Template, error) {
	return template.ParseFiles(templatePath+"layout.html", templatePath+"login.html")
}

func login(w http.ResponseWriter, r *http.Request) {

	tmpl, err := getTemplates()

	if err != nil {
		http.Error(w, "Error loading templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:    "Log in",
		Template: "login",
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}

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

func loadTemplates(files ...string) (*template.Template, error) {
	var paths []string
	for _, file := range files {
		paths = append(paths, templatePath+file)
	}

	return template.ParseFiles(paths...)
}

func apiLogin(w http.ResponseWriter, r *http.Request) {

	tmpl, err := loadTemplates("layout.html", "login.html")
	if err != nil {
		log.Printf("Template loading error: %v", err)
		http.Error(w, "Error loading templates", http.StatusInternalServerError)
		return
	}

	err = r.ParseForm()
	if err != nil {
		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Invalid username or password",
		}
		w.WriteHeader(http.StatusInternalServerError)
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Template execution error: %v", err)
			return
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Checks that username and password are not empty strings
	if username == "" || password == "" {

		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Username and password cannot be empty",
		}
		w.WriteHeader(http.StatusInternalServerError)
		err := tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	var user User

	// Finds the user in the db based on the username
	err = db.QueryRow("SELECT id, username, password FROM users WHERE username = $1", username).Scan(&user.ID, &user.Username, &user.Password)

	// If the username is not found in th db or password is incorrect
	if err != nil {
		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Incorrect username or password",
		}
		w.WriteHeader(http.StatusInternalServerError)
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Template execution error: %v", err)
			return
		}
		return
	}

	if !validatePassword(user.Password, password) {
		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Incorrect username or password",
		}

		w.WriteHeader(http.StatusInternalServerError)
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Template execution error: %v", err)
			return
		}
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Session error",
		}
		w.WriteHeader(http.StatusInternalServerError)
		err := tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	session.Values["user_id"] = user.ID
	if err := session.Save(r, w); err != nil {
		data := PageData{
			Title:    "Log in",
			Template: "login.html",
			Error:    "Session save error",
		}
		w.WriteHeader(http.StatusInternalServerError)
		err := tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

//////////////////////////////////////////////////////////////////////////////////
/// Register handlers
//////////////////////////////////////////////////////////////////////////////////

// viser registreringssiden.
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if userIsLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	tmpl, err := template.ParseFiles(templatePath + "register.html")
	if err != nil {
		http.Error(w, "Error loading register page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// Håndterer registreringsprocessen
func apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if userIsLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusFound)
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

	var userID int
	err = db.QueryRow("INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id",
		username, email, hashedPassword).Scan(&userID)

	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create session and log the user in
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = int(userID)
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Session save error", http.StatusInternalServerError)
		return
	}

	// Omdiriger til forsiden (bruger er nu logget ind)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Ser om brugere allerede eksisterer
func userExists(username, email string) (bool, bool) {
	var usernameExists, emailExists bool

	//Use a transaction to ensure consistent reads
	tx, err := db.Begin()
	if err != nil {
		return false, false
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Rollback error: %v", err)
		}
	}()

	// Tjekker for eksisterende brugernavn
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1))", username).Scan(&usernameExists)
	if err != nil {
		return false, false
	}

	// Tjekker for eksisterende email
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))", email).Scan(&emailExists)
	if err != nil {
		return false, false
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Commit error: %v", err)
		return false, false
	}
	return usernameExists, emailExists
}

// tjekker om brugeren er logget in

func userIsLoggedIn(r *http.Request) bool {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return false
	}

	// Check if user_id exists in session
	userID, ok := session.Values["user_id"]
	return ok && userID != nil
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

//////////////////////////////////////////////////////////////////////////////////
/// Security Functions
//////////////////////////////////////////////////////////////////////////////////

// hasher passwordet med bcrypt.
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

// startCronScheduler sets up and starts a cron job that runs checkTables() every minute
func startCronScheduler() {

	c := cron.New()

	// Schedule the checkTables function to run every minute
	// Cron expression "*/1 * * * *" means it runs at the start of every minute
	_, err := c.AddFunc("*/1 * * * *", func() {
		fmt.Println("Cron job: Running checkTables at", time.Now())
		checkTables()
	})
	if err != nil {
		log.Fatalf("Error scheduling cron job: %v", err)
	}

	c.Start()

}

//////////////////////////////////////////////////////////////////////////////////
/// Main
//////////////////////////////////////////////////////////////////////////////////

func main() {
	// initialiserer databasen og forbinder til den.
	initDB()
	defer closeDB()

	//Initialize Elasticsearch
	initElasticsearch()

	if err := syncPagesToElasticsearch(); err != nil {
		log.Fatalf("Failed to sync pages: %v", err)
	}

	checkTables()

	// Start the cron scheduler to run checkTables periodically
	startCronScheduler()

	// Detter er Gorilla Mux's route handler, i stedet for Flasks indbyggede router-handler
	///Opretter en ny router
	r := mux.NewRouter()

	//Definerer routerne.
	r.HandleFunc("/", rootHandler).Methods("GET")             // Forside
	r.HandleFunc("/about", aboutHandler).Methods("GET")       //about-side
	r.HandleFunc("/login", login).Methods("GET")              //Login-side
	r.HandleFunc("/register", registerHandler).Methods("GET") //Register-side
	r.HandleFunc("/search", searchHandler).Methods("GET")

	// Definerer api-erne
	r.HandleFunc("/api/login", apiLogin).Methods("POST")
	r.HandleFunc("/api/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("POST") // API-ruten for søgninger.
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	r.HandleFunc("/api/weather", weatherHandler).Methods("GET") //weather-side

	// sørger for at vi kan bruge de statiske filer som ligger i static-mappen. ex: css.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

}
