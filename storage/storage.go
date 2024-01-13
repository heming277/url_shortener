// storage/storage.go
package storage

import (
	"database/sql"
	"url-shortener/models"
)

var db *sql.DB

// SaveUser saves a new user to the PostgreSQL database.
func SaveUser(user models.User) error {
	// SQL query to insert a new user
	query := `INSERT INTO users (id, email, password) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, user.ID, user.Email, user.Password)
	return err
}

// GetUserByEmail retrieves a user by email from the PostgreSQL database.
func GetUserByEmail(email string) (models.User, error) {
	var user models.User
	// SQL query to fetch the user by email
	query := `SELECT id, email, password FROM users WHERE email = $1`
	err := db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password)
	return user, err
}

// SaveURLMapping saves a new URL mapping to the PostgreSQL database.
func SaveURLMapping(urlMapping models.URLMapping) error {
	// SQL query to insert a new URL
	query := `INSERT INTO urls (user_id, original_url, shortened_url) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, urlMapping.UserID, urlMapping.OriginalURL, urlMapping.ShortCode)
	return err
}

// GetUserURLMappings retrieves all URL mappings for a user from the PostgreSQL database.
func GetUserURLMappings(userID string) ([]models.URLMapping, error) {
	var urlMappings []models.URLMapping
	// SQL query to fetch all URLs for the user
	query := `SELECT shortened_url, original_url FROM urls WHERE user_id = $1`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var urlMapping models.URLMapping
		if err := rows.Scan(&urlMapping.ShortCode, &urlMapping.OriginalURL); err != nil {
			return nil, err
		}
		urlMappings = append(urlMappings, urlMapping)
	}
	return urlMappings, nil
}

// GetURLMappingByOriginalURL retrieves a URL mapping by original URL and user ID from the PostgreSQL database.
func GetURLMappingByOriginalURL(userID, originalURL string) (models.URLMapping, error) {
	var urlMapping models.URLMapping
	query := `SELECT shortened_url FROM urls WHERE user_id = $1 AND original_url = $2`
	err := db.QueryRow(query, userID, originalURL).Scan(&urlMapping.ShortCode)
	if err != nil {
		return urlMapping, err
	}
	urlMapping.OriginalURL = originalURL
	urlMapping.UserID = userID
	return urlMapping, nil
}

// GetURLMappingByShortCode retrieves a URL mapping by the short code.
func GetURLMappingByShortCode(shortCode string) (models.URLMapping, error) {
	var urlMapping models.URLMapping
	query := `SELECT user_id, original_url FROM urls WHERE shortened_url = $1`
	err := db.QueryRow(query, shortCode).Scan(&urlMapping.UserID, &urlMapping.OriginalURL)
	if err != nil {
		return urlMapping, err
	}
	urlMapping.ShortCode = shortCode
	return urlMapping, nil
}

// IncrementURLVisitCount increments the visit count for a URL.
func IncrementURLVisitCount(userID, shortCode string) error {
	query := `UPDATE urls SET visit_count = visit_count + 1 WHERE user_id = $1 AND shortened_url = $2`
	_, err := db.Exec(query, userID, shortCode)
	return err
}
