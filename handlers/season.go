package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"skyhawk/db"
	"strconv"
	"strings"

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

	} else if stat == "minutes" {
		query := fmt.Sprintf(`
			SELECT match_id, stat, minute
			FROM matches_stats
			WHERE %s = $1 AND EXTRACT(YEAR FROM match_date) = $2
		`, idColumn)

		rows, err := db.PG.Query(query, entityId, season)
		if err != nil {
			log.Printf("Error querying minutes: %v", err)
			http.Error(w, "Error querying minutes", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type event struct {
			matchID int
			stat    string
			minute  string
		}

		eventsByMatch := make(map[int][]event)

		for rows.Next() {
			var ev event
			if err := rows.Scan(&ev.matchID, &ev.stat, &ev.minute); err != nil {
				log.Printf("Row scan error: %v", err)
				continue
			}
			if ev.stat == "in" || ev.stat == "out" {
				eventsByMatch[ev.matchID] = append(eventsByMatch[ev.matchID], ev)
			}
		}

		totalSeconds := 0.0

		for _, events := range eventsByMatch {
			var matchSeconds float64
			var stack []float64

			for _, ev := range events {
				parts := strings.Split(ev.minute, ".")
				var min, sec int
				if len(parts) == 1 {
					min, _ = strconv.Atoi(parts[0]) // Only minutes, seconds default to 0
					sec = 0
				} else if len(parts) == 2 {
					min, _ = strconv.Atoi(parts[0]) // Minutes
					sec, _ = strconv.Atoi(parts[1]) // Seconds
				} else {
					continue
				}
				t := float64(min*60 + sec)

				if ev.stat == "in" {
					stack = append(stack, t)
				} else if ev.stat == "out" && len(stack) > 0 {
					inTime := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if t > inTime {
						matchSeconds += t - inTime
					}
				}
			}

			totalSeconds += matchSeconds
		}

		avg = totalSeconds / float64(matchCount) / 60.0

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
