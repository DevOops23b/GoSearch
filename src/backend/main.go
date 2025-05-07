package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	log.Printf("CONN_STR: %s", CONN_STR)
	// initialiserer databasen og forbinder til den.
	initDB()
	defer closeDB()

	// Wait for database to be fully ready
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if err := db.Ping(); err != nil {
			log.Printf("Database not ready yet, retrying in 2 seconds... (%d/%d)", i+1, maxRetries)
			time.Sleep(2 * time.Second)
		} else {
			log.Println("Database connection confirmed!")
			break
		}

		if i == maxRetries-1 {
			log.Fatalf("Failed to connect to database after %d attempts", maxRetries)
		}
	}

	err := setupPasswordResetTable()
	if err != nil {
		log.Printf("Warning: Password reset setup had errors: %v", err)
		log.Println("Will attempt to continue startup anyway...")
	} else {
		log.Println("Password reset functionality successfully initialized")
	}

	//!!!Only comment in if all passwords of all users needs to be reset!!!

	/*if err := forceResetForAllUsers(); err != nil {
		log.Printf("Warning: Failed to force password reset for all users: %v", err)
	} else {
		log.Println("Successfully forced all users to reset their passwords")
	}*/

	//Initialize Elasticsearch
	initElasticsearch()

	if err := syncPagesToElasticsearch(); err != nil {
		log.Fatalf("Failed to sync pages: %v", err)
	}

	///NEW: initialize searchLogger////
	logPath := os.Getenv("SEARCH_LOG_PATH")
	if logPath == "" {
		logPath = "search.log" // Default for Docker
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: could not open search log file: %v, using stdout instead", err)
		searchLogger = log.New(os.Stdout, "SEARCH: ", log.LstdFlags)
	} else {
		log.Printf("Search logs will be written to %s", logPath)
		searchLogger = log.New(f, "SEARCH: ", log.LstdFlags)
		defer f.Close()
	}

	checkTables()

	// Start the cron scheduler to run checkTables periodically
	startCronScheduler()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	fmt.Println("Database connection successful!")

	startMonitoring()

	// Detter er Gorilla Mux's route handler, i stedet for Flasks indbyggede router-handler
	///Opretter en ny router
	r := mux.NewRouter()
	r.Use(passwordResetMiddleware)

	fmt.Println("Registering /metrics endpoint...")
	r.Handle("/metrics", promhttp.Handler())

	// Applying middleware function to all routes

	appRouter := r.NewRoute().Subrouter()
	appRouter.Use(metricsMiddleware)

	//Definerer routerne.
	r.HandleFunc("/", rootHandler).Methods("GET")             // Forside
	r.HandleFunc("/about", aboutHandler).Methods("GET")       //about-side
	r.HandleFunc("/login", login).Methods("GET")              //Login-side
	r.HandleFunc("/register", registerHandler).Methods("GET") //Register-side
	r.HandleFunc("/search", searchHandler).Methods("GET")
	r.HandleFunc("/reset-password", resetPasswordHandler).Methods("GET")

	// Definerer api-erne
	r.HandleFunc("/api/login", apiLogin).Methods("POST")
	r.HandleFunc("/api/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("POST") // API-ruten for søgninger.
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	r.HandleFunc("/api/weather", weatherHandler).Methods("GET") //weather-side
	r.HandleFunc("/api/reset-password", apiResetPasswordHandler).Methods("POST")

	// sørger for at vi kan bruge de statiske filer som ligger i static-mappen. ex: css.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

}
