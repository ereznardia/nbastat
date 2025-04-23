package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"skyhawk/db"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func GetAverageStat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	season := vars["season"]
	entity := vars["entity"]
	entityId := vars["entityId"]
	stat := vars["stat"]

	var avg float64
	var idColumn string

	switch entity {
	case "player":
		idColumn = "player_id"
	case "team":
		idColumn = "team_id"
	default:
		http.Error(w, "Wrong entity type", http.StatusBadRequest)
		log.Printf("Wrong entity type: %s", entity)
		return
	}

	// Get total distinct matches played
	matchCount := 0
	err := db.PG.QueryRow(fmt.Sprintf(`
		SELECT COUNT(DISTINCT match_id)
		FROM matches_stats
		WHERE %s = $1 AND EXTRACT(YEAR FROM match_date) = $2
	`, idColumn), entityId, season).Scan(&matchCount)
	if err != nil || matchCount == 0 {
		http.Error(w, "No matches found or error", http.StatusInternalServerError)
		log.Printf("Match count error: %v", err)
		return
	}

	getStatCount := func(statName string) int {
		var count int
		err := db.PG.QueryRow(fmt.Sprintf(`
			SELECT COUNT(*)
			FROM matches_stats
			WHERE %s = $1 AND stat = $2 AND EXTRACT(YEAR FROM match_date) = $3
		`, idColumn), entityId, statName, season).Scan(&count)
		if err != nil {
			log.Printf("Stat count error for %s: %v", statName, err)
			return 0
		}
		return count
	}

	if stat == "points" {
		points := []struct {
			name  string
			value float64
		}{
			{"1pt", 1},
			{"2pt", 2},
			{"3pt", 3},
		}
		var total float64
		for _, pt := range points {
			total += float64(getStatCount(pt.name)) * pt.value
		}
		avg = total / float64(matchCount)
	} else {
		avg = float64(getStatCount(stat)) / float64(matchCount)
	}

	response := map[string]interface{}{
		"entityId": entityId,
		"stat":     stat,
		"season":   season,
		"average":  avg,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
