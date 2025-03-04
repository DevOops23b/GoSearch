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


// Test function signature:
// func TestName(t *testing.T)

// Mock user for testing
/*
type mockDB struct {}

type mockRow struct {
    error error
}

func (r *mockRow) Scan(dest...interface{}) error {
    return r.error
}



func (mdb *mockDB) QueryRow(query string, args ... interface{}) *mockRow {
    if args[0] == "validUser" && args[1] == "validPassword" {
        return &mockRow{error: nil}
    }
    return &mockRow{error: errors.New("Nothing to find here")}
}

// Login
func TestLoginSuccess(t *testing.T) {
    store := sessions.NewCookieStore([]byte("test-secret"))
    requestBody := map[string]string{
        "username": "validUser",
        "password": "validPassword",
    }
    body, _ := json.Marshal(requestBody)
    request := httptest.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
    request.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    ApiLogin(w, request)

    res := w.Result()

    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200 but got %v", res.StatusCode)
    }

    cookie := res.Cookies()
    if len(cookie) == 0 {
        t.Errorf("Expected to find a cookie, but found none")
    }

}

// Test failed login

func TestLoginFailed(t *testing.T) {
    store := sessions.NewCookieStore([]byte("test-secret"))
    requestBody := map[string]string {
        "username": "invalidUser",
        "password": "wrongPassword",
    }


}
*/

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

/*
// Failed login
func TestFailedLogin(t *testing.T) {
    testCases := []struct {
        name        string
        username    string
        password    string


    }
}
    */




