// api/router.go
package api

import (
	"net/http"
	"url-shortener/handlers"
	"github.com/gorilla/mux"
	"url-shortener/storage"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter()

	// Define the API endpoints and map them to handlers
	router.HandleFunc("/create", handlers.CreateShortURLHandler).Methods("POST").Handler(storage.RateLimitMiddleware(http.HandlerFunc(handlers.CreateShortURLHandler)))
	router.HandleFunc("/{shortCode}", handlers.RedirectShortURLHandler).Methods("GET")
	router.HandleFunc("/analytics/{shortCode}", handlers.GetURLAnalyticsHandler).Methods("GET")

	router.HandleFunc("/signup", handlers.SignUpHandler).Methods("POST")
	router.HandleFunc("/login", handlers.LoginHandler).Methods("POST")

	router.HandleFunc("/user/urls", handlers.GetUserURLsHandler).Methods("GET")
	router.HandleFunc("/delete/{shortCode}", handlers.DeleteURLHandler).Methods("DELETE")
	router.HandleFunc("/urls/{shortCode}/visit", handlers.AuthenticatedVisitCountHandler).Methods("POST")

	router.HandleFunc("/user/urls/{shortCode}/visitcount", handlers.GetURLVisitCountHandler).Methods("GET")

	return router
}