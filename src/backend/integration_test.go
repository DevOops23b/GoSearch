
package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

// setupRouter duplicates main() route setup for testing.
func setupRouter() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.HandleFunc("/about", aboutHandler).Methods("GET")
	r.HandleFunc("/api/weather", weatherHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("GET")
	r.HandleFunc("/api/login", apiLogin).Methods("POST")
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	return r
}

// setupTestDB initializes an in-memory SQLite DB and schema.
func setupTestDB(t *testing.T) {
	var err error
	// overwrite the package-level db variable
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	schema := `
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT,
    email TEXT,
    password TEXT
);
CREATE TABLE pages (
    title TEXT,
    url TEXT,
    language TEXT,
    last_updated DATETIME,
    content TEXT
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
}

func TestWeatherHandler(t *testing.T) {
	testName := "Weather"
	t.Run(testName, func(t *testing.T) {
		setupTestDB(t)
		store = sessions.NewCookieStore([]byte("test-secret"))

		h := setupRouter()
		ts := httptest.NewServer(h)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/weather?city=Copenhagen")
		if err != nil {
			t.Fatalf("%s request failed: %v", testName, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s expected 200 OK, got %d", testName, resp.StatusCode)
		}

		body, _ := ioutil.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Copenhagen") {
			t.Errorf("%s expected 'Copenhagen' in body, got %s", testName, string(body))
		}
	})
}

func TestRootHandler(t *testing.T) {
	testName := "Root"
	t.Run(testName, func(t *testing.T) {
		setupTestDB(t)
		store = sessions.NewCookieStore([]byte("test-secret"))

		// ensure templates folder exists
		tmplDir := filepath.Join("frontend", "templates")
		if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
			t.Skip("templates not found, skipping root handler test")
		}

		h := setupRouter()
		ts := httptest.NewServer(h)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("%s request failed: %v", testName, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s expected 200 OK, got %d", testName, resp.StatusCode)
		}

		body, _ := ioutil.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Home") {
			t.Errorf("%s expected 'Home' in body, got %s", testName, string(body))
		}
	})
}

func TestSearchHandler(t *testing.T) {
	testName := "Search"
	t.Run(testName, func(t *testing.T) {
		setupTestDB(t)
		// seed a page in the in-memory DB
		_, err := db.Exec(`INSERT INTO pages (title,url,language,last_updated,content) VALUES (?,?,?,?,?);`,
			"TestTitle", "/test-url", "en", "2025-01-01", "TestContent")
		if err != nil {
			t.Fatalf("%s failed to seed pages: %v", testName, err)
		}

		store = sessions.NewCookieStore([]byte("test-secret"))
		h := setupRouter()
		ts := httptest.NewServer(h)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/search?q=TestContent&language=en")
		if err != nil {
			t.Fatalf("%s request failed: %v", testName, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s expected 200 OK, got %d", testName, resp.StatusCode)
		}

		body, _ := ioutil.ReadAll(resp.Body)
		if !strings.Contains(string(body), "TestTitle") {
			t.Errorf("%s expected 'TestTitle' in response, got %s", testName, string(body))
		}
	})
}

func TestAPILoginHandler(t *testing.T) {
	testName := "API Login"
	t.Run(testName, func(t *testing.T) {
		setupTestDB(t)
		// seed a user
		hashed, _ := hashPassword("pass123")
		_, err := db.Exec(`INSERT INTO users (username,email,password) VALUES (?,?,?);`,
			"user1", "u1@example.com", hashed)
		if err != nil {
			t.Fatalf("%s failed to seed user: %v", testName, err)
		}

		store = sessions.NewCookieStore([]byte("test-secret"))
		h := setupRouter()
		ts := httptest.NewServer(h)
		defer ts.Close()

		// use a client that does not follow redirects
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		resp, err := client.PostForm(ts.URL+"/api/login", url.Values{
			"username": {"user1"},
			"password": {"pass123"},
		})
		if err != nil {
			t.Fatalf("%s POST failed: %v", testName, err)
		}
		defer resp.Body.Close()

		// expect redirect status on success
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%s expected 303 SeeOther, got %d", testName, resp.StatusCode)
		}
	})
}

func TestAPIRegisterHandler(t *testing.T) {
	testName := "API Register"
	t.Run(testName, func(t *testing.T) {
		setupTestDB(t)
		store = sessions.NewCookieStore([]byte("test-secret"))
		h := setupRouter()
		ts := httptest.NewServer(h)
		defer ts.Close()

		// use client without redirects
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		resp, err := client.PostForm(ts.URL+"/api/register", url.Values{
			"username":  {"newuser"},
			"email":     {"new@example.com"},
			"password":  {"abc123"},
			"password2": {"abc123"},
		})
		if err != nil {
			t.Fatalf("%s POST failed: %v", testName, err)
		}
		defer resp.Body.Close()

		// expect redirect status on success
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%s expected 303 SeeOther, got %d", testName, resp.StatusCode)
		}
	})
}
