package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"skyhawk/db"
	"skyhawk/routes"
)

func main() {
	// Read from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	db.InitPostgres(fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName))
	db.InitRedis("localhost:6379", "", 0)

	r := routes.SetupRouter()
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
