// handlers/handlers_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"url-shortener/storage"

	"github.com/gorilla/mux"
)

func TestMain(m *testing.M) {

	redisClient = storage.NewRedisClient()
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestCreateAndRedirectShortURL(t *testing.T) {

	router := mux.NewRouter()
	router.HandleFunc("/create", CreateShortURLHandler).Methods("POST")
	router.HandleFunc("/{shortCode}", RedirectShortURLHandler).Methods("GET")

	// Test creating a short URL.
	payload := map[string]string{
		"originalUrl": "https://example.com",
	}
	jsonPayload, _ := json.Marshal(payload)

	createReq, _ := http.NewRequest("POST", "/create", bytes.NewBuffer(jsonPayload))
	createReq.Header.Set("Content-Type", "application/json")

	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if status := createRR.Code; status != http.StatusOK {
		t.Errorf("CreateShortURLHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var createResult map[string]string
	err := json.Unmarshal(createRR.Body.Bytes(), &createResult)
	if err != nil {
		t.Fatalf("could not unmarshal response from create: %v", err)
	}

	shortCode, ok := createResult["shortCode"]
	if !ok {
		t.Fatalf("CreateShortURLHandler response does not contain 'shortCode'")
	}

	// Store the short URL
	err = redisClient.StoreURLMapping(shortCode, payload["originalUrl"], 24*time.Hour)
	if err != nil {
		t.Fatalf("could not store URL mapping in Redis: %v", err)
	}

	redirectReq, _ := http.NewRequest("GET", "/"+shortCode, nil)
	redirectRR := httptest.NewRecorder()
	router.ServeHTTP(redirectRR, redirectReq)

	// Get result
	result := redirectRR.Result()
	defer result.Body.Close()

	// Check status
	if status := result.StatusCode; status != http.StatusFound {
		t.Errorf("RedirectShortURLHandler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	// Check that the Location header is set to the original URL.
	location, ok := result.Header["Location"]
	if !ok {
		t.Fatal("Location header is not set in the response")
	}

	if location[0] != payload["originalUrl"] {
		t.Errorf("RedirectShortURLHandler returned wrong Location header: got %v want %v", location[0], payload["originalUrl"])
	}
}
