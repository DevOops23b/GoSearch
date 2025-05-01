package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

var (
	searchLogger *log.Logger
	staticPath   = "static"
)

var CONN_STR string

var templatePath string

var db *sql.DB

var esClient *elasticsearch.Client

var store = sessions.NewCookieStore([]byte("Very-secret-key"))

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
