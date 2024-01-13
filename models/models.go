// models/models.go
package models

// URLMapping represents the structure of the URL storage.
type URLMapping struct {
	UserID      string `json:"userId,omitempty"`
	ShortCode   string `json:"shortCode"`
	OriginalURL string `json:"originalUrl"`
	VisitCount  int    `json:"visitCount"`
}

type User struct {
	ID       string
	Email    string
	Password string // hashed password
}
