package main

import (
	"net/http"
	"sync"
)

// Maps to track active sessions
var (
	activeSessions     = make(map[string]bool)
	activeSessionsLock sync.Mutex
)

// Increments the user sessions counter when a new session is created
func incrementUserSessionsTotal(authStatus string) {
	userSessionsTotal.WithLabelValues(authStatus).Inc()
}

// Records a user request with authentication status
func recordUserRequest(r *http.Request, authStatus string) {
	userRequestsTotal.WithLabelValues(authStatus, r.URL.Path).Inc()
}

// Adds a session to the active sessions tracking
func trackActiveSession(sessionID string, authStatus string) {
	activeSessionsLock.Lock()
	defer activeSessionsLock.Unlock()

	// Only increment counter if this is a new session
	if _, exists := activeSessions[sessionID]; !exists {
		activeSessions[sessionID] = true
		activeUserSessions.WithLabelValues(authStatus).Inc()
	}
}

// Removes a session from active sessions tracking
func removeActiveSession(sessionID string, authStatus string) {
	activeSessionsLock.Lock()
	defer activeSessionsLock.Unlock()

	if _, exists := activeSessions[sessionID]; exists {
		delete(activeSessions, sessionID)
		// Decrement the counter for this auth status
		activeUserSessions.WithLabelValues(authStatus).Dec()
	}
}

// Returns the authentication status for the current request
func getAuthStatus(r *http.Request) string {
	// Get the session from the request
	session, err := store.Get(r, "session-name")
	if err != nil {
		return "anonymous"
	}

	// Check if the user is authenticated
	if userID, ok := session.Values["user_id"]; ok && userID != nil {
		return "authenticated"
	}
	return "anonymous"
}
