package routes

import (
	"skyhawk/handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	// General data routes - Using Postgres for data structuring and historic data, season aggregatios, complex player stats etc

	r.HandleFunc("/api/tables", handlers.DeletePgTables).Methods("DELETE") // delete all tables -- todo: delete it

	r.HandleFunc("/api/table/{tableName}", handlers.DeletePgTable).Methods("DELETE")

	r.HandleFunc("/api/teams", handlers.GetTeams).Methods("GET")  // Get all teams
	r.HandleFunc("/api/teams", handlers.AddTeams).Methods("POST") // Add multiple teams

	r.HandleFunc("/api/player", handlers.DeletePlayer).Methods("DELETE") // Delete a player by ID
	r.HandleFunc("/api/players", handlers.GetPlayers).Methods("GET")     // Get all players
	r.HandleFunc("/api/players", handlers.AddPlayers).Methods("POST")    // Add multiple players

	r.HandleFunc("/api/player_team_history", handlers.GetPlayerTeamHistories).Methods("GET")  // Get player team history
	r.HandleFunc("/api/player_team_history", handlers.AddPlayerTeamHistories).Methods("POST") // Add player team history

	r.HandleFunc("/api/team_active_players/{teamId}", handlers.GetTeamActivePlayers).Methods("GET") // Get team active players

	r.HandleFunc("/api/matches", handlers.GetMatches).Methods("GET")  // Get team match history
	r.HandleFunc("/api/matches", handlers.AddMatches).Methods("POST") // Add team match history

	// Live match routes - Using Redis for real time performance
	r.HandleFunc("/api/match_stat", handlers.AddMatchStat).Methods("POST")
	r.HandleFunc("/api/match_stats", handlers.GetMatchStats).Methods("GET")
	r.HandleFunc("/api/match_stat/{matchId}", handlers.GetMatchStat).Methods("GET")
	r.HandleFunc("/api/match_stat/{matchId}/{playerId}", handlers.GetPlayerMatchStat).Methods("GET")

	r.HandleFunc("/api/start_match/{matchId}", handlers.StartMatch).Methods("POST")

	return r
}
