package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// ForcePasswordReset adds a column to track if password has been changed
func setupPasswordResetTable() error {
	log.Println("Setting up password reset functionality...")

	// Ensure we can connect to the database
	if err := db.Ping(); err != nil {
		log.Printf("Error connecting to database during password reset setup: %v", err)
		return err
	}

	// Check if the password_changed column exists in the users table
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'password_changed'
		)
	`).Scan(&exists)

	if err != nil {
		log.Printf("Error checking for password_changed column: %v", err)
		return err
	}

	if !exists {
		log.Println("password_changed column does not exist, adding it now...")
		// Add password_changed column to the users table
		_, err := db.Exec(`
			ALTER TABLE users
			ADD COLUMN password_changed BOOLEAN DEFAULT TRUE
		`)
		if err != nil {
			log.Printf("Error adding password_changed column: %v", err)
			return err
		}

		log.Println("Successfully added password_changed column to users table")

		// Update existing users to have password_changed = true
		_, err = db.Exec(`UPDATE users SET password_changed = FALSE`)
		if err != nil {
			log.Printf("Error updating existing users' password_changed status: %v", err)
			return err
		}
		log.Println("Successfully updated existing users to have password_changed = FALSE")
	} else {
		log.Println("password_changed column already exists")
	}

	// Check if the reset_tokens table exists
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_name = 'reset_tokens'
		)
	`).Scan(&exists)

	if err != nil {
		log.Printf("Error checking for reset_tokens table: %v", err)
		return err
	}

	if !exists {
		log.Println("reset_tokens table does not exist, creating it now...")
		// Create reset_tokens table
		_, err := db.Exec(`
			CREATE TABLE reset_tokens (
				user_id INTEGER PRIMARY KEY REFERENCES users(id),
				token TEXT NOT NULL,
				expires_at TIMESTAMP NOT NULL
			)
		`)
		if err != nil {
			log.Printf("Error creating reset_tokens table: %v", err)
			return err
		}
		log.Println("Successfully created reset_tokens table")
	} else {
		log.Println("reset_tokens table already exists")
	}

	// Verify the setup was successful
	if err := verifySetup(); err != nil {
		return err
	}

	return nil
}

// New function to verify the password reset setup
func verifySetup() error {
	log.Println("Verifying password reset setup...")

	// Check password_changed column
	var passwordChangedExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'password_changed'
		)
	`).Scan(&passwordChangedExists)

	if err != nil {
		log.Printf("Error verifying password_changed column: %v", err)
		return err
	}

	if passwordChangedExists {
		log.Println("✅ password_changed column exists")
	} else {
		log.Println("❌ password_changed column DOES NOT exist")
		return err
	}

	// Check reset_tokens table
	var resetTokensExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_name = 'reset_tokens'
		)
	`).Scan(&resetTokensExists)

	if err != nil {
		log.Printf("Error verifying reset_tokens table: %v", err)
		return err
	}

	if resetTokensExists {
		log.Println("✅ reset_tokens table exists")
	} else {
		log.Println("❌ reset_tokens table DOES NOT exist")
		return err
	}

	return nil
}

// ResetPasswordHandler displays the reset password form
func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Printf("Session error in resetPasswordHandler: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"]
	if !ok || userID == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check if password has already been changed
	var passwordChanged bool
	err = db.QueryRow("SELECT password_changed FROM users WHERE id = $1", userID).Scan(&passwordChanged)
	if err != nil {
		log.Printf("Database error in resetPasswordHandler: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// If password already changed, redirect to home
	if passwordChanged {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get username for display
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)
	if err != nil {
		log.Printf("Error fetching username: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(templatePath + "resetPassword.html")
	if err != nil {
		log.Printf("Error loading reset password template: %v", err)
		http.Error(w, "Error loading reset password page", http.StatusInternalServerError)
		return
	}

	data := struct {
		UserID   int
		Username string
		Error    string
	}{
		UserID:   userID.(int),
		Username: username,
		Error:    "",
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// apiResetPasswordHandler handles the password reset POST request
func apiResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Printf("Session error in apiResetPasswordHandler: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"]
	if !ok || userID == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	err = r.ParseForm()
	if err != nil {
		log.Printf("Form parsing error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get form values
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Check if passwords match
	if newPassword != confirmPassword {
		renderResetPasswordError(w, r, userID.(int), "New passwords do not match")
		return
	}

	// Validate new password (add more validation as needed)
	if len(newPassword) < 8 {
		renderResetPasswordError(w, r, userID.(int), "New password must be at least 8 characters long")
		return
	}

	// Get current user's password hash
	var currentPasswordHash string
	err = db.QueryRow("SELECT password FROM users WHERE id = $1", userID).Scan(&currentPasswordHash)
	if err != nil {
		log.Printf("Error fetching password hash: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Verify current password
	if !validatePassword(currentPasswordHash, currentPassword) {
		renderResetPasswordError(w, r, userID.(int), "Current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Update user's password and set password_changed to true
	_, err = db.Exec("UPDATE users SET password = $1, password_changed = TRUE WHERE id = $2", hashedPassword, userID)
	if err != nil {
		log.Printf("Error updating password: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Log successful password reset
	log.Printf("User ID %d successfully reset their password", userID)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// renderResetPasswordError renders the reset password page with an error message
func renderResetPasswordError(w http.ResponseWriter, r *http.Request, userID int, errorMsg string) {
	tmpl, err := template.ParseFiles(templatePath + "resetPassword.html")
	if err != nil {
		log.Printf("Error loading reset password template: %v", err)
		http.Error(w, "Error loading reset password page", http.StatusInternalServerError)
		return
	}

	// Get username for display
	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)
	if err != nil {
		log.Printf("Error fetching username: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := struct {
		UserID   int
		Username string
		Error    string
	}{
		UserID:   userID,
		Username: username,
		Error:    errorMsg,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// checkPasswordResetRequired checks if user needs to reset their password
// Returns true if password reset is required
func checkPasswordResetRequired(userID int) bool {
	var passwordChanged bool

	// First check if the column exists to avoid errors
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'password_changed'
		)
	`).Scan(&columnExists)

	if err != nil {
		log.Printf("Error checking if password_changed column exists: %v", err)
		return false
	}

	if !columnExists {
		log.Printf("password_changed column does not exist, trying to add it")
		_, err := db.Exec("ALTER TABLE users ADD COLUMN password_changed BOOLEAN DEFAULT TRUE")
		if err != nil {
			log.Printf("Failed to add password_changed column: %v", err)
			return false
		}
		log.Printf("Successfully added password_changed column")
		return true
	}

	// Now check the actual value
	err = db.QueryRow("SELECT password_changed FROM users WHERE id = $1", userID).Scan(&passwordChanged)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User %d not found", userID)
			return false
		}
		log.Printf("Error checking password reset: %v", err)
		return false
	}

	return !passwordChanged
}

// passwordResetMiddleware checks if the user needs to reset their password
// and redirects them to the reset page if necessary
func passwordResetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip middleware for these paths
		if r.URL.Path == "/reset-password" ||
			r.URL.Path == "/api/reset-password" ||
			r.URL.Path == "/login" ||
			r.URL.Path == "/api/login" ||
			r.URL.Path == "/register" ||
			r.URL.Path == "/api/register" ||
			r.URL.Path == "/api/logout" ||
			strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// Check if user is logged in
		session, err := store.Get(r, "session-name")
		if err != nil {
			log.Printf("Session error in passwordResetMiddleware: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		userID, ok := session.Values["user_id"]
		if !ok || userID == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check if password reset is required
		if checkPasswordResetRequired(userID.(int)) {
			log.Printf("Password reset required for user %d, redirecting to reset page", userID)
			http.Redirect(w, r, "/reset-password", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Function to force reset for all users (admin function)
func forceResetForAllUsers() error {
	_, err := db.Exec("UPDATE users SET password_changed = FALSE")
	if err != nil {
		log.Printf("Error forcing password reset for all users: %v", err)
		return err
	}
	log.Println("Successfully forced password reset for all users")
	return nil
}
