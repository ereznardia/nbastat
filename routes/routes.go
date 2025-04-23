package routes

import (
	"skyhawk/handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	// General data routes - Using Postgres for data structuring and historic data, season aggregatios, complex player stats etc

	r.HandleFunc("/api/teams", handlers.GetTeams).Methods("GET")  // Get all teams
	r.HandleFunc("/api/teams", handlers.AddTeams).Methods("POST") // Add multiple teams

	r.HandleFunc("/api/player", handlers.DeletePlayer).Methods("DELETE") // Delete a player by ID
	r.HandleFunc("/api/players", handlers.GetPlayers).Methods("GET")     // Get all players
	r.HandleFunc("/api/players", handlers.AddPlayers).Methods("POST")    // Add multiple players

	r.HandleFunc("/api/player_team_history", handlers.GetPlayerTeamHistories).Methods("GET")  // Get all players team history
	r.HandleFunc("/api/player_team_history", handlers.AddPlayerTeamHistories).Methods("POST") // Add players team history
	r.HandleFunc("/api/leave_team", handlers.LeaveTeam).Methods("POST")                       // Add players team history

	r.HandleFunc("/api/team_active_players/{teamId}", handlers.GetTeamActivePlayers).Methods("GET") // Get team active players

	r.HandleFunc("/api/matches", handlers.GetMatches).Methods("GET")  // Get team match history
	r.HandleFunc("/api/matches", handlers.AddMatches).Methods("POST") // Add team match history

	//******************************//
	//**** seasonal match stats ****//
	//******************************//
	// taken from 'matches_stats' (it will be populated in the end of the live match stat system, once match is over)

	r.HandleFunc("/api/player_stats/{playerId}/{season_year}", handlers.GetPlayerSeasonStats).Methods("GET")
	// r.HandleFunc("/api/player_stats", handlers.GetMatchStats).Methods("GET")

	// Live match routes - Using Redis for real time performance
	r.HandleFunc("/api/match_stat", handlers.AddMatchStat).Methods("POST")
	r.HandleFunc("/api/match_stats", handlers.GetMatchStats).Methods("GET")
	r.HandleFunc("/api/match_stat/{matchId}/{entity}/{entityId}", handlers.GetMatchStat).Methods("GET")

	r.HandleFunc("/api/start_match/{matchId}", handlers.StartMatch).Methods("POST")
	r.HandleFunc("/api/end_match/{matchId}", handlers.EndMatch).Methods("POST")

	return r
}
