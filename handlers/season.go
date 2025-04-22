package handlers

import (
	"encoding/json"
	"net/http"
	"skyhawk/db"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func GetPlayerSeasonStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	playerID, err := strconv.Atoi(vars["playerId"])
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	seasonYear := vars["season_year"]
	statFilter := r.URL.Query().Get("stat")
	if statFilter == "" {
		http.Error(w, "Missing required 'stat' query parameter", http.StatusBadRequest)
		return
	}

	rows, err := db.PG.Query(`
		SELECT ms.match_id, ms.player_id, ms.minute, ms.stat
		FROM matches_stats ms
		JOIN matches m ON ms.match_id = m.match_id
		WHERE ms.player_id = $1
		AND ms.stat = $2
		AND EXTRACT(YEAR FROM m.date) = $3
	`, playerID, statFilter, seasonYear)
	if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stats []map[string]interface{}

	for rows.Next() {
		var matchID, playerID int
		var minute float64
		var stat string

		if err := rows.Scan(&matchID, &playerID, &minute, &stat); err != nil {
			http.Error(w, "Failed to parse row", http.StatusInternalServerError)
			return
		}

		stats = append(stats, map[string]interface{}{
			"match_id":  matchID,
			"player_id": playerID,
			"minute":    minute,
			"stat":      stat,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
