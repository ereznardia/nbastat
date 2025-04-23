package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var PG *sql.DB

func InitPostgres(dsn string) {
	var err error
	PG, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	if err = PG.Ping(); err != nil {
		log.Fatalf("Postgres not reachable: %v", err)
	}

	log.Println("Postgres connected")

	// After connection, check for tables existence.
	// In real enviroment, i would have some infrastructure tool to initialize and maintain the infrastructure (Terraform or something)
	// For now, I will check tables here and create if not there.
	checkForTable()

}

func checkForTable() {
	// Create Tables if they don't exist
	createTableIfNotExists("teams", `
		CREATE TABLE teams (
			team_id SERIAL PRIMARY KEY,  -- Auto-incrementing ID
			team_name TEXT NOT NULL UNIQUE
		);
	`)

	// PG.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", "players"))
	createTableIfNotExists("players", `
			CREATE TABLE players (
				player_id SERIAL PRIMARY KEY,  -- Auto-incrementing ID
				first_name TEXT NOT NULL,
				last_name TEXT NOT NULL
			);
		`)

	// PG.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", "player_team_history"))
	createTableIfNotExists("player_team_history", `
		CREATE TABLE player_team_history (
			history_id SERIAL PRIMARY KEY,  -- Auto-incrementing ID
			player_id INT REFERENCES players(player_id) ON DELETE CASCADE,
			team_id INT REFERENCES teams(team_id) ON DELETE CASCADE,
			start_date DATE NOT NULL,
			end_date DATE
		);
	`)

	if _, err := PG.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_player_team_history
		ON player_team_history (player_id, team_id, COALESCE(end_date, DATE '9999-12-31'));
	`); err != nil {
		log.Fatalf("Error creating unique player-team-history index: %v", err)
	}

	// PG.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", "matches"))
	createTableIfNotExists("matches", `
			CREATE TABLE matches (
				match_id SERIAL PRIMARY KEY,  -- Auto-incrementing ID
				date DATE NOT NULL,
				home_team INT REFERENCES teams(team_id) ON DELETE CASCADE,
				away_team INT REFERENCES teams(team_id) ON DELETE CASCADE,
				home_score INT,
				away_score INT
			);
		`)

	if _, err := PG.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_match_date
		ON matches (date, home_team, away_team);
	`); err != nil {
		log.Fatalf("Error creating unique match index: %v", err)
	}

	// PG.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", "matches_stats"))
	createTableIfNotExists("matches_stats", `
			CREATE TABLE matches_stats (
				match_id INT REFERENCES matches(match_id) ON DELETE CASCADE,
				team_id INT REFERENCES teams(team_id) ON DELETE CASCADE,
				player_id INT REFERENCES players(player_id) ON DELETE CASCADE,
				minute REAL,
				stat TEXT NOT NULL,
				match_date DATE NOT NULL
			);
		`)
}

func createTableIfNotExists(tableName, createSQL string) {
	var exists bool
	err := PG.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1
		);
	`, tableName).Scan(&exists)

	if err != nil {
		log.Fatalf("Error checking if table exists: %v", err)
	}

	if !exists {
		_, err := PG.Exec(createSQL)
		if err != nil {
			log.Fatalf("Error creating table %s: %v", tableName, err)
		} else {
			fmt.Printf("Table %s created successfully.\n", tableName)
		}
	} else {
		fmt.Printf("Table %s already exists.\n", tableName)
	}
}
