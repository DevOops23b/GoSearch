package main

import (
	"testing"
    "net/http"
    "net/http/httptest"
    "database/sql"
    "github.com/gorilla/sessions"
    "net/url"
    "strings"
    "golang.org/x/crypto/bcrypt"
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
)

// Helper function to create a mock database
func setupMockDB() (*sql.DB, sqlmock.Sqlmock) {
    mockDB, mock, err := sqlmock.New()
    if err != nil {
        panic(err)
    }

    db = mockDB
    return mockDB, mock
}

func TestLoginSuccess(t *testing.T) {
    mockDB, mock := setupMockDB()
    defer mockDB.Close()

    testUsername := "testUser"
    testPassword := "testPassword"

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
    assert.NoError(t, err)

    mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
        WithArgs(testUsername).WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password"}).
        AddRow(1, testUsername, string(hashedPassword)))

    mockStore := sessions.NewCookieStore([]byte("test-secret"))
    store = mockStore

    formData := url.Values {
        "username": {testUsername},
        "password": {testPassword},
    }

    req, err := http.NewRequest("POST", "/api/login", strings.NewReader(formData.Encode()))
    assert.NoError(t, err)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    w := httptest.NewRecorder()

    apiLogin(w, req)

    response := w.Result()

    assert.Equal(t, http.StatusSeeOther, response.StatusCode, "Expected redirect after successful login")

}


// Failed login
func TestFailedLogin(t *testing.T) {
    testCases := []struct {
        name            string
        username        string
        password        string
        mockSetup       func(mock sqlmock.Sqlmock)
        expectedStatus  int
    } {
        {
        name:           "Empty Username",
        username:       "",
        password:       "testPassword",
        mockSetup:      func(mock sqlmock.Sqlmock) {},
        expectedStatus: http.StatusInternalServerError,
    },
    {
        name:           "Empty password",
        username:       "testUser",
        password:       "",
        mockSetup:      func(mock sqlmock.Sqlmock) {},
        expectedStatus: http.StatusInternalServerError,
    },
    {
        name:           "Incorrect password",
        username:       "testUser",
        password:       "invalidPassword",
        mockSetup:      func(mock sqlmock.Sqlmock) {
            hashedPassword, err := bcrypt.GenerateFromPassword([]byte("validPassword"), bcrypt.DefaultCost)
            mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").WithArgs("testUser").
            
        },
        expectedStatus: 
    }
}
}




