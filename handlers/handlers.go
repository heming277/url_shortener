// handlers/handlers.go
package handlers

import (
	"fmt"
    "net/url"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"url-shortener/models"
	"url-shortener/storage"
	"url-shortener/utils"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
    "strings"
	//"github.com/ory/hydra-client-go/client"
)

var redisClient *storage.RedisClient


/*var hydraAdminURL = "https://exciting-tesla-xohu7lej80.projects.oryapis.com"
var hydraClient = client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
	Schemes:  []string{"http"},
	Host:     hydraAdminURL,
	BasePath: "/",
})*/

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

	// Initiate the Ory Cloud OAuth2 authorization flow
	authURL := "https://exciting-tesla-xohu7lej80.projects.oryapis.com/oauth2/auth"
	clientID := "c674a060-44f9-404f-9488-46cd12c6b6fb"
	redirectURI := "http://localhost:8080/callback"
	scope := "openid offline_access"

	authQuery := url.Values{}
	authQuery.Set("client_id", clientID)
	authQuery.Set("redirect_uri", redirectURI)
	authQuery.Set("scope", scope)
	authQuery.Set("response_type", "code")

	authURLWithQuery := fmt.Sprintf("%s?%s", authURL, authQuery.Encode())

	// Redirect the user to the Ory Cloud authorization endpoint
	http.Redirect(w, r, authURLWithQuery, http.StatusTemporaryRedirect)

	// For now just return success message
	w.Write([]byte("Login successful"))
}


func CallbackHandler(w http.ResponseWriter, r *http.Request) {
    // Get the authorization code from the query parameters
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "Authorization code is missing", http.StatusBadRequest)
        return
    }

    // Exchange the authorization code for an access token and refresh token
    tokenURL := "https://exciting-tesla-xohu7lej80.projects.oryapis.com/oauth2/token"
    clientID := "c674a060-44f9-404f-9488-46cd12c6b6fb"
    //clientSecret := "" // Not required since the client authentication method is set to "none"
    redirectURI := "http://localhost:8080/callback"

    tokenRequest, err := http.NewRequest("POST", tokenURL, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    tokenQuery := url.Values{}
    tokenQuery.Set("grant_type", "authorization_code")
    tokenQuery.Set("code", code)
    tokenQuery.Set("client_id", clientID)
    tokenQuery.Set("redirect_uri", redirectURI)

    tokenRequest.Body = ioutil.NopCloser(strings.NewReader(tokenQuery.Encode()))

    client := &http.Client{}
    tokenResponse, err := client.Do(tokenRequest)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer tokenResponse.Body.Close()

    var tokenData struct {
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        // Other token data fields
    }

    err = json.NewDecoder(tokenResponse.Body).Decode(&tokenData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Store the access token and refresh token in the session or cookie
    // You can also perform additional logic, such as redirecting the user to a protected page

    fmt.Fprintf(w, "Access Token: %s\nRefresh Token: %s", tokenData.AccessToken, tokenData.RefreshToken)

	http.Redirect(w, r, "/", http.StatusFound)
}