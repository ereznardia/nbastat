package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"skyhawk/db"
	"time"

	"database/sql"
	"log"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func DeletePgTables(w http.ResponseWriter, r *http.Request) {
	// List of tables you created
	tables := []string{
		"teams",
		"players",
		"player_team_history",
		"matches",
		"matches_stats",
	}

	// Start a transaction to ensure all operations happen atomically
	tx, err := db.PG.Begin()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Ensure rollback in case of failure

	// Loop through each table name and drop the table
	for _, tableName := range tables {
		// Drop the table if it exists
		_, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
		if err != nil {
			http.Error(w, fmt.Sprintf("Error dropping table %s: %v", tableName, err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Table %s dropped successfully.\n", tableName)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("Error committing transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Return a success message
	w.WriteHeader(http.StatusOK)
	response := "Specified tables have been deleted successfully."
	w.Write([]byte(response))
}

func DeletePgTable(w http.ResponseWriter, r *http.Request) {
	// Extract the table name from the URL parameters
	vars := mux.Vars(r)
	tableName := vars["tableName"]

	// Validate the table name to prevent SQL injection
	if tableName == "" {
		http.Error(w, "Table name is required", http.StatusBadRequest)
		return
	}

	// Prepare the SQL query to drop the table
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)

	// Execute the query
	_, err := db.PG.Exec(sql)
	if err != nil {
		http.Error(w, "Failed to delete table", http.StatusInternalServerError)
		log.Printf("Error deleting table %s: %v", tableName, err)
		return
	}

	// Send a success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Table '%s' deleted successfully", tableName)))
}

// TEAMS APIS //

func GetTeams(w http.ResponseWriter, r *http.Request) {
	// Query to select all teams from the database
	rows, err := db.PG.Query(`
		SELECT team_id, team_name FROM teams
	`)

	if err != nil {
		// Handle error if the query fails
		http.Error(w, fmt.Sprintf("Error selecting teams: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close() // Ensure rows are closed when done

	// Create a slice to hold the teams
	var teams []map[string]interface{}

	// Loop through the rows and fetch team data
	for rows.Next() {
		var teamID int64 // Change to int64 as the team_id is now an integer
		var teamName string

		// Scan the row into variables
		if err := rows.Scan(&teamID, &teamName); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}

		// Add the team to the result slice
		teams = append(teams, map[string]interface{}{
			"team_id":   teamID, // Return the int64 team_id
			"team_name": teamName,
		})
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Respond with the teams data in JSON format
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(teams); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %v", err), http.StatusInternalServerError)
		return
	}
}

func AddTeams(w http.ResponseWriter, r *http.Request) {
	var teams []map[string]string
	err := json.NewDecoder(r.Body).Decode(&teams)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	for _, team := range teams {
		teamName := team["teamName"]

		// Insert team without specifying team_id (auto-increment will take care of it)
		_, err := db.PG.Exec(`
			INSERT INTO teams (team_name)
			VALUES ($1)
		`, teamName)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error adding team %s: %v", teamName, err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	response := fmt.Sprintf("Teams added successfully.")
	w.Write([]byte(response))
}

// PLAYERS APIS //

func DeletePlayer(w http.ResponseWriter, r *http.Request) {
	type RequestBody struct {
		PlayerID string `json:"playerId"` // The player_id field from the JSON body
	}

	// Read the body of the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unmarshal the JSON body into the RequestBody struct
	var requestData RequestBody
	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	// Ensure that player_id is provided in the body
	if requestData.PlayerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	// Query to delete the player from the database
	_, err = db.PG.Exec(`
		DELETE FROM players WHERE player_id = $1`, requestData.PlayerID)

	if err != nil {
		// Handle error if the query fails
		http.Error(w, fmt.Sprintf("Error deleting player: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Send a successful response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "Player %s deleted successfully"}`, requestData.PlayerID)))
}

func GetPlayers(w http.ResponseWriter, r *http.Request) {
	// Query to select all players from the database
	rows, err := db.PG.Query(`
		SELECT player_id, first_name, last_name FROM players
	`)

	if err != nil {
		// Handle error if the query fails
		http.Error(w, fmt.Sprintf("Error selecting players: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close() // Ensure rows are closed when done

	// Create a slice to hold the players
	var players []map[string]interface{}

	// Loop through the rows and fetch player data
	for rows.Next() {
		var playerID int64 // Change to int64 as player_id is an integer
		var firstName, lastName string

		// Scan the row into variables
		if err := rows.Scan(&playerID, &firstName, &lastName); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}

		// Add the player to the result slice
		players = append(players, map[string]interface{}{
			"player_id":  playerID, // Return player_id as an integer
			"first_name": firstName,
			"last_name":  lastName,
		})
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Respond with the players data in JSON format
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(players); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %v", err), http.StatusInternalServerError)
		return
	}
}

func AddPlayers(w http.ResponseWriter, r *http.Request) {
	var players []map[string]string
	err := json.NewDecoder(r.Body).Decode(&players)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	for _, player := range players {
		firstName := player["firstName"]
		lastName := player["lastName"]

		// Insert player without specifying player_id (auto-increment will take care of it)
		_, err := db.PG.Exec(`
			INSERT INTO players (first_name, last_name)
			VALUES ($1, $2)
		`, firstName, lastName)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error adding player %s %s: %v", firstName, lastName, err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Players added successfully."))
}

func GetPlayerTeamHistories(w http.ResponseWriter, r *http.Request) {
	type PlayerHistory struct {
		PlayerFullName string       `json:"playerFullName"`
		TeamName       string       `json:"teamName"`
		StartDate      string       `json:"startDate"`
		EndDate        sql.NullTime `json:"endDate"`
	}

	// Query the database
	rows, err := db.PG.Query(`
		SELECT CONCAT(p.first_name, ' ', p.last_name), t.team_name, pth.start_date, pth.end_date
		FROM player_team_history pth
		JOIN teams t ON pth.team_id = t.team_id
		JOIN players p ON pth.player_id = p.player_id
	`)
	if err != nil {
		http.Error(w, "Failed to query database", http.StatusInternalServerError)
		log.Printf("Error querying database: %v", err)
		return
	}
	defer rows.Close()

	// Initialize a slice to hold the player team history data
	var playerHistory []PlayerHistory

	// Iterate over the rows returned by the query and append to the playerHistory slice
	for rows.Next() {
		var history PlayerHistory

		// Scan the data into the history struct
		if err := rows.Scan(&history.PlayerFullName, &history.TeamName, &history.StartDate, &history.EndDate); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			log.Printf("Error scanning row: %v", err)
			return
		}

		// Append the history to the slice
		playerHistory = append(playerHistory, history)
	}

	// Check for any errors encountered during row iteration
	if err := rows.Err(); err != nil {
		http.Error(w, "Error processing database rows", http.StatusInternalServerError)
		log.Printf("Error iterating over rows: %v", err)
		return
	}

	// Set the content-type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Respond with the player team history as a JSON array
	if len(playerHistory) > 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(playerHistory)
	} else {
		// If no history was found for the player, respond with an empty array
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]struct{}{})
	}
}

func AddPlayerTeamHistories(w http.ResponseWriter, r *http.Request) {
	var playerHistory []struct {
		PlayerID  int    `json:"playerId"`
		TeamID    int    `json:"teamId"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
	}

	// Parse the incoming JSON into a flat structure
	if err := json.NewDecoder(r.Body).Decode(&playerHistory); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Insert each record into the database
	for _, history := range playerHistory {
		var endDate any
		if history.EndDate == "" {
			endDate = nil
		} else {
			endDate = history.EndDate
		}

		sql := `INSERT INTO player_team_history (player_id, team_id, start_date, end_date) VALUES ($1, $2, $3, $4)`
		_, err := db.PG.Exec(sql,
			history.PlayerID, history.TeamID, history.StartDate, endDate)

		if err != nil {
			http.Error(w, "Failed to insert player team history", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func GetMatches(w http.ResponseWriter, r *http.Request) {
	// Query to select all matches from the database
	rows, err := db.PG.Query(`
		SELECT match_id, date, home_team, away_team FROM matches
	`)

	if err != nil {
		// Handle error if the query fails
		http.Error(w, fmt.Sprintf("Error selecting matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close() // Ensure rows are closed when done

	// Create a slice to hold the matches
	var matches []map[string]interface{}

	// Loop through the rows and fetch player data
	for rows.Next() {
		var matchID int64
		var date time.Time
		var homeTeam int64
		var awayTeam int64

		// Scan the row into variables
		if err := rows.Scan(&matchID, &date, &homeTeam, &awayTeam); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}

		// Add the player to the result slice
		matches = append(matches, map[string]interface{}{
			"match_id":  matchID,
			"date":      date,
			"home_team": homeTeam,
			"away_team": awayTeam,
		})
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Respond with the players data in JSON format
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(matches); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %v", err), http.StatusInternalServerError)
		return
	}
}

func AddMatches(w http.ResponseWriter, r *http.Request) {
	var matches []map[string]string
	err := json.NewDecoder(r.Body).Decode(&matches)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	for _, match := range matches {
		date := match["date"]
		homeTeam := match["homeTeam"]
		awayTeam := match["awayTeam"]

		if date == "" || homeTeam == "" || awayTeam == "" {
			http.Error(w, "Missing fields in match data", http.StatusBadRequest)
			return
		}

		_, err := db.PG.Exec(`
			INSERT INTO matches (date, home_team, away_team)
			VALUES ($1,$2,$3)
		`, date, homeTeam, awayTeam)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error adding match %s vs %s at $s: %v", homeTeam, awayTeam, date), http.StatusInternalServerError)
			return
		}

		log.Printf("Match added: %s - Home: %s vs Away: %s", date, homeTeam, awayTeam)
	}

	// Respond to client
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Matches added successfully")
}
