package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"url-shortener/api"
	"url-shortener/handlers"
	"url-shortener/storage"

	"github.com/rs/cors"
)

func main() {

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	dbConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	storage.InitDB(dbConnString)

	router := api.NewRouter()

	// Set up CORS options
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allows all origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
		Debug:            true,
	})

	// Apply the CORS middleware to the router
	handler := corsHandler.Handler(router)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/consent", handlers.ConsentHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
