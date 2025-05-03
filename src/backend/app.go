package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// initialiserer databasen og forbinder til den.
	initDB()
	defer closeDB()

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

	// Applying middleware function to all routes
	appRouter := r.NewRoute().Subrouter()
	appRouter.Use(metricsMiddleware)

	//Definerer routerne.
	appRouter.HandleFunc("/", rootHandler).Methods("GET")             // Forside
	appRouter.HandleFunc("/about", aboutHandler).Methods("GET")       //about-side
	appRouter.HandleFunc("/login", login).Methods("GET")              //Login-side
	appRouter.HandleFunc("/register", registerHandler).Methods("GET") //Register-side
	appRouter.HandleFunc("/search", searchHandler).Methods("GET")

	// Definerer api-erne
	appRouter.HandleFunc("/api/login", apiLogin).Methods("POST")
	appRouter.HandleFunc("/api/logout", logoutHandler).Methods("GET")
	appRouter.HandleFunc("/api/search", searchHandler).Methods("GET")
	appRouter.HandleFunc("/api/search", searchHandler).Methods("POST") // API-ruten for søgninger.
	appRouter.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	appRouter.HandleFunc("/api/weather", weatherHandler).Methods("GET") //weather-side

	// sørger for at vi kan bruge de statiske filer som ligger i static-mappen. ex: css.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	fmt.Println("Registering /metrics endpoint...")
	r.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server running on http://localhost:8080")
	//Starter serveren.
	log.Fatal(http.ListenAndServe(":8080", r))

}
