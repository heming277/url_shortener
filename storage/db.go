// storage/db.go
package storage

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func InitDB(dataSourceName string) {
	var err error
	db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to the database: %q", err)
	}

	fmt.Println("Connected to the database successfully!")
}
