// handlers/handlers.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"url-shortener/models"
	"url-shortener/storage"
	"url-shortener/utils"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var redisClient *storage.RedisClient

func init() {
	// Initialize the Redis client
	redisClient = storage.NewRedisClient()
}

// CreateShortURLHandler handles requests for creating short URLs.
func CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
	var urlMapping models.URLMapping
	// Decode the incoming JSON payload
	if err := json.NewDecoder(r.Body).Decode(&urlMapping); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Sanitize the original URL
	sanitizedURL, err := utils.SanitizeURL(urlMapping.OriginalURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	urlMapping.OriginalURL = sanitizedURL

	var userID string
	isNew := false

	if userID != "" {
		existingMapping, err := storage.GetURLMappingByOriginalURL(userID, urlMapping.OriginalURL)
		if err == nil {
			// URL already exists for this user, return the existing short code
			urlMapping.ShortCode = existingMapping.ShortCode
		} else {
			// URL does not exist, create a new short code and store it in PostgreSQL
			urlMapping.ShortCode = utils.GenerateRandomString(8)
			urlMapping.UserID = userID
			if err := storage.SaveURLMapping(urlMapping); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		// For guests, check if the URL already exists in Redis
		existingShortCode, err := redisClient.GetShortCodeByURL(urlMapping.OriginalURL)
		if err != nil && !errors.Is(err, storage.ErrURLNotFound) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if existingShortCode == "" {
			// Generate a short code for the URL
			urlMapping.ShortCode = utils.GenerateRandomString(8)
			// Store the URL mapping in Redis with a 24-hour expiration
			if err := redisClient.StoreURLMapping(urlMapping.ShortCode, urlMapping.OriginalURL, 24*time.Hour); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			isNew = true
		} else {
			// Use the existing short code
			urlMapping.ShortCode = existingShortCode
		}
	}
	// Respond with the short URL and the isNew flag
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		OriginalURL string `json:"originalUrl"`
		ShortCode   string `json:"shortCode"`
		IsNew       bool   `json:"isNew"`
	}{
		OriginalURL: urlMapping.OriginalURL,
		ShortCode:   urlMapping.ShortCode,
		IsNew:       isNew,
	}
	json.NewEncoder(w).Encode(response)
}

// RedirectShortURLHandler handles requests for redirecting to the original URL.
func RedirectShortURLHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := mux.Vars(r)["shortCode"]

	// Attempt to retrieve the original URL from PostgreSQL first
	urlMapping, err := storage.GetURLMappingByShortCode(shortCode)
	if err == nil {
		// URL found in PostgreSQL, increment the visit count there
		if err := storage.IncrementURLVisitCount(urlMapping.UserID, shortCode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Redirect to the original URL
		http.Redirect(w, r, urlMapping.OriginalURL, http.StatusFound)
		return
	}

	// If the URL is not found in PostgreSQL, check Redis (for guests)
	originalURL, err := redisClient.RetrieveOriginalURL(shortCode)
	if err != nil {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	// Increment the visit count in Redis
	if err := redisClient.IncrementVisitCount(shortCode); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the original URL
	http.Redirect(w, r, originalURL, http.StatusFound)
}

// GetURLAnalyticsHandler handles requests for getting URL analytics.
func GetURLAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := mux.Vars(r)["shortCode"]
	count, err := redisClient.GetVisitCount(shortCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"visitCount": count})
}

func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	// Save the user to the database
	err = storage.SaveUser(user)
	if err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "user created"})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string
		Password string
	}
	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Authenticate the user
	user, err := storage.GetUserByEmail(credentials.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check the password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// need to create an OAuth 2.0 authorization request with ORY Hydra
	// redirecting the user to the Hydra login consent flow,
	// set a cookie or session to remember the user's login state
	// and then redirect them to the Hydra authorization URL.

	// For now just return success message
	w.Write([]byte("Login successful"))
}

func ConsentHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Render consent page or handle consent logic
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Handle the callback from Hydra
	// This is where you would handle the authorization code or tokens from Hydra
}
