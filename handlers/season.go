package handlers

import (
	"encoding/json"
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
			var query string
			if entity == "player" {
				query = `
					SELECT COUNT(*) * 1.0 / COUNT(DISTINCT match_id) AS avg
					FROM matches_stats
					WHERE player_id = $1 AND stat = $2 AND EXTRACT(YEAR FROM match_date) = $3
				`
			} else if entity == "team" {
				query = `
					SELECT COUNT(*) * 1.0 / COUNT(DISTINCT match_id) AS avg
					FROM matches_stats
					WHERE team_id = $1 AND stat = $2 AND EXTRACT(YEAR FROM match_date) = $3
				`
			} else {
				http.Error(w, "Wrong entity type", http.StatusBadRequest)
				log.Printf("Wrong entity type: %s", entity)
				return
			}

			var ptAvg float64
			err := db.PG.QueryRow(query, entityId, pt.name, season).Scan(&ptAvg)
			if err != nil {
				log.Printf("Average stat query error for %s: %v", pt.name, err)
			}

			total += ptAvg * pt.value
		}

		avg = total
	} else {
		var query string
		if entity == "player" {
			query = `
				SELECT COUNT(*) * 1.0 / COUNT(DISTINCT match_id) AS avg
				FROM matches_stats
				WHERE player_id = $1 AND stat = $2 AND EXTRACT(YEAR FROM match_date) = $3
			`
		} else if entity == "team" {
			query = `
				SELECT COUNT(*) * 1.0 / COUNT(DISTINCT match_id) AS avg
				FROM matches_stats
				WHERE team_id = $1 AND stat = $2 AND EXTRACT(YEAR FROM match_date) = $3
			`
		} else {
			http.Error(w, "Wrong entity type", http.StatusBadRequest)
			log.Printf("Wrong entity type: %s", entity)
			return
		}

		err := db.PG.QueryRow(query, entityId, stat, season).Scan(&avg)
		if err != nil {
			http.Error(w, "Error calculating average stat", http.StatusInternalServerError)
			log.Printf("Average stat query error: %v", err)
			return
		} else {
			log.Printf("ok")
		}
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
