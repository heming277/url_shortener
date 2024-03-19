// models/models.go
package models

// URLMapping represents the structure of the URL storage.
type URLMapping struct {
	UserID      int    `json:"userId"`
	ShortCode   string `json:"shortCode"`
	OriginalURL string `json:"originalUrl"`
	VisitCount  int    `json:"visitCount"`
}

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"` // hashed password
}