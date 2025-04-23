package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"skyhawk/db"
	"skyhawk/routes"
	"strconv"
	"time"
)

func startRedisKeyLogger() {
	ticker := time.NewTicker(3 * time.Second)

	go func() {
		for range ticker.C {
			// Fetch Redis keys that match the pattern "match:*"
			keys, err := db.Redis.Keys(db.Ctx, "match:*").Result()

			// Check for errors while fetching keys
			if err != nil {
				log.Printf("[Redis Logger] Error fetching match keys: %v", err)
				continue
			}

			// If no keys found, log an appropriate message
			if len(keys) == 0 {
				log.Println("[Redis Logger] No active match keys found.")
			} else {
				// Log the found keys
				log.Printf("[Redis Logger] Active match keys: %v", keys)
			}
		}
	}()
}

func clearAllMatchStats() error {
	var cursor uint64
	match := "match:*"

	for {
		keys, nextCursor, err := db.Redis.Scan(db.Ctx, cursor, match, 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %v", err)
		}

		if len(keys) > 0 {
			if err := db.Redis.Del(db.Ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func main() {
	// startRedisKeyLogger()

	// Env vars
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

	// clearAllMatchStats()

	r := routes.SetupRouter()

	//
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/", http.StripPrefix("/", fs))

	http.Handle("/api/", r)

	// Start the server
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
