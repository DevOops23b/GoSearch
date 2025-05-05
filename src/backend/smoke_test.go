package main

import (
	"testing"
)

func TestSmoke_DatabaseConnection(t *testing.T) {
	// Prøv at connecte til DB
	db, err := connectDB()
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Check at Ping() også virker
	if err := db.Ping(); err != nil {
		t.Fatalf("DB ping failed: %v", err)
	}
}
