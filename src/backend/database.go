package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"os"

	"github.com/robfig/cron/v3"
)

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
		err := rows2.Scan(&page.URL, &page.Title, &page.Language, &page.LastUpdated, &page.Content)
		if err != nil {
			log.Printf("Error scanning page: %v", err)
			continue
		}
		fmt.Printf("Title: %s, URL: %s, Language: %s\n", page.Title, page.URL, page.Language)
	}
}

func startCronScheduler() {
    c := cron.New()
    
    // Schedule the checkTables function to run every minute
    _, err := c.AddFunc("*/1 * * * *", func() {
        fmt.Println("Cron job: Running checkTables at", time.Now())
        checkTables()
    })
    if err != nil {
        log.Fatalf("Error scheduling cron job: %v", err)
    }
    
    // scraping wikipedia every 5. minutes
    c.AddFunc("*/5 * * * *", func() {
        fmt.Println("Cron job: Running Wikipedia scraper at", time.Now())
        logPath := os.Getenv("SEARCH_LOG_PATH")
        if logPath == "" {
            logPath = "search.log"
        }
        
        // Track the number of pages before scraping
        var countBefore int
        err := db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&countBefore)
        if err != nil {
            log.Printf("Error getting page count before scraping: %v", err)
        }
        
        // Run scraping
        StartScraping(logPath)
        
        // Check if new pages were added
        var countAfter int
        err = db.QueryRow("SELECT COUNT(*) FROM pages").Scan(&countAfter)
        if err != nil {
            log.Printf("Error getting page count after scraping: %v", err)
        }
        
        // Only sync to Elasticsearch if new pages were added
        if countAfter > countBefore {
            log.Printf("New pages added (%d -> %d). Syncing to Elasticsearch.", countBefore, countAfter)
            err := syncPagesToElasticsearch()
            if err != nil {
                log.Printf("Error syncing to Elasticsearch: %v", err)
            } else {
                log.Println("Synced scraped pages to Elasticsearch successfully.")
            }
        } else {
            log.Println("No new pages added. Skipping Elasticsearch sync.")
        }
    })
    
    c.Start()
}
