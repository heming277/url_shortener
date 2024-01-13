// api/router.go
package api

import (
	//"net/http"

	"url-shortener/handlers"

	"github.com/gorilla/mux"

	"url-shortener/storage"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(storage.RateLimitMiddleware)
	// Define the API endpoints and map them to handlers
	router.HandleFunc("/create", handlers.CreateShortURLHandler).Methods("POST")
	router.HandleFunc("/{shortCode}", handlers.RedirectShortURLHandler).Methods("GET")
	router.HandleFunc("/analytics/{shortCode}", handlers.GetURLAnalyticsHandler).Methods("GET")

	return router
}
