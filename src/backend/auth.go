package main

import (
	"database/sql"
	"log"
	"strings"

	//"encoding/json" // Gør at vi kan læse json-format
	"net/http" // til http-servere og håndtering af routere

	// en router til http-requests

	//Til at hashe passwords
	"golang.org/x/crypto/bcrypt"

	// Database-connection. Go undersøtter ikke SQLite, og derfor skal vi importere en driver
	_ "github.com/mattn/go-sqlite3"
	// Import the cron library to schedule periodic tasks
	// PostgreSQL driver instead of SQLite.
	_ "github.com/lib/pq"
)


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
