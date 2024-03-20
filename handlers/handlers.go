// handlers/handlers.go
package handlers

import (

	"encoding/json"
	"errors"
	"net/http"
	"time"
	"strings"
	"url-shortener/models"
	"url-shortener/storage"
	"url-shortener/utils"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"github.com/dgrijalva/jwt-go"
	"log"

)

var redisClient *storage.RedisClient
var jwtKey = []byte("+iQmsWxcpcHN+YPHUojt9iVgBtsrhPm59cR9q1+F4Lk=")

type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func init() {
	// Initialize the Redis client
	redisClient = storage.NewRedisClient()
}

func GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
    email, err := getEmailFromToken(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    user, err := storage.GetUserByEmail(email)
    if err != nil {
        log.Printf("Error retrieving user by email: %v", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    urlMappings, err := storage.GetUserURLMappings(user.ID)
    if err != nil {
        log.Printf("Error retrieving URL mappings: %v", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(urlMappings)
}


func getEmailFromToken(r *http.Request) (string, error) {
    tokenString := r.Header.Get("Authorization")
    if tokenString == "" {
        return "", errors.New("authorization header is missing")
    }
    tokenString = strings.TrimPrefix(tokenString, "Bearer ")

    claims := &Claims{}

    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return jwtKey, nil
    })

    if err != nil || !token.Valid {
        return "", errors.New("invalid token")
    }

    return claims.Email, nil
}

// CreateShortURLHandler handles requests for creating short URLs.
func CreateShortURLHandler(w http.ResponseWriter, r *http.Request) {
    var urlMapping models.URLMapping
    if err := json.NewDecoder(r.Body).Decode(&urlMapping); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    sanitizedURL, err := utils.SanitizeURL(urlMapping.OriginalURL)
    if err != nil {
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }
    urlMapping.OriginalURL = sanitizedURL

    email, err := getEmailFromToken(r)
    isNew := false

    if err == nil && email != "" {
        user, err := storage.GetUserByEmail(email)
        if err != nil {
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        existingMapping, err := storage.GetURLMappingByOriginalURL(user.ID, urlMapping.OriginalURL)
        if err == nil {
            urlMapping.ShortCode = existingMapping.ShortCode
        } else {
            urlMapping.ShortCode = utils.GenerateRandomString(8)
            urlMapping.UserID = user.ID
            if err := storage.SaveURLMapping(urlMapping); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            isNew = true
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
        VisitCount  int    `json:"visitCount"`
    }{
        OriginalURL: urlMapping.OriginalURL,
        ShortCode:   urlMapping.ShortCode,
        IsNew:       isNew,
        VisitCount:  0, // Initialize the visit count to 0 for new URLs
    }
    json.NewEncoder(w).Encode(response)
}

// RedirectShortURLHandler handles requests for redirecting to the original URL.
func RedirectShortURLHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := mux.Vars(r)["shortCode"]

	// Attempt to retrieve the original URL from PostgreSQL first
	urlMapping, err := storage.GetURLMappingByShortCode(shortCode)
	if err == nil {
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

func DeleteURLHandler(w http.ResponseWriter, r *http.Request) {
    shortCode := mux.Vars(r)["shortCode"]
    email, err := getEmailFromToken(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    user, err := storage.GetUserByEmail(email)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    err = storage.DeleteURLMapping(user.ID, shortCode)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}



func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	err = storage.SaveUser(user)
	if err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// Create the JWT token for the newly registered user
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
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

	user, err := storage.GetUserByEmail(credentials.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: credentials.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}


func AuthenticatedVisitCountHandler(w http.ResponseWriter, r *http.Request) {
    // Get the email from the token
    email, err := getEmailFromToken(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    // Get the user by email
    user, err := storage.GetUserByEmail(email)
    if err != nil {
        log.Printf("Error retrieving user by email: %v", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Get the short code from the URL path
    vars := mux.Vars(r)
    shortCode := vars["shortCode"]

    // Increment the visit count in the database
    err = storage.IncrementURLVisitCount(user.ID, shortCode)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return a success message
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Visit count incremented successfully"))

}

func GetURLVisitCountHandler(w http.ResponseWriter, r *http.Request) {
    // Get the email from the token
    email, err := getEmailFromToken(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    // Get the user by email
    user, err := storage.GetUserByEmail(email)
    if err != nil {
        log.Printf("Error retrieving user by email: %v", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Get the short code from the URL path
    vars := mux.Vars(r)
    shortCode := vars["shortCode"]

    // Get the visit count from the database
    count, err := storage.GetURLVisitCount(user.ID, shortCode)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return the visit count as JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]int{"visitCount": count})
}
