package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"skyhawk/db"
	"skyhawk/routes"
	"strconv"
)

func main() {
	// Read from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := os.Getenv("REDIS_DB")

	if redisDBStr == "" {
		redisDBStr = "0"
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Fatalf("Invalid REDIS_DB value: %v", err)
	}

	db.InitPostgres(fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName))
	db.InitRedis(redisAddr, redisPassword, redisDB)

	r := routes.SetupRouter()
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
