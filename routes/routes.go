package routes

import (
	"handlers"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	// General data routes
	r.HandleFunc("/api/teams", handlers.CreateTeam).Methods("POST")
	r.HandleFunc("/api/players", handlers.CreatePlayer).Methods("POST")

	// Live match routes
	r.HandleFunc("/api/match/live-update", handlers.LiveMatchUpdate).Methods("POST")

	return r
}
