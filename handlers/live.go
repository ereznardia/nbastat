package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"skyhawk/db"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var validStatsToAdd = []string{
	"rebounds", "assists", "steals", "blocks", "turnovers",
	"fouls", "in", "out", "1pt", "2pt", "3pt",
}

var validStatsToFetch = []string{
	"rebounds", "assists", "steals", "blocks", "turnovers",
	"fouls", "minutes", "1pt", "2pt", "3pt", "points",
}

var pointValues = map[string]int{
	"1pt": 1,
	"2pt": 2,
	"3pt": 3,
}

func AddMatchStat(w http.ResponseWriter, r *http.Request) {
	var MatchStat struct {
		MatchID  int    `json:"matchId"`
		PlayerID int    `json:"playerId"`
		Minute   string `json:"minute"`
		Stat     string `json:"stat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&MatchStat); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidMinuteValue(MatchStat.Minute) {
		http.Error(w, "Invalid minute value", http.StatusBadRequest)
		return
	}

	if !slices.Contains(validStatsToAdd, MatchStat.Stat) {
		http.Error(w, fmt.Sprintf("Invalid stat type. Available stats to add are: %v", validStatsToAdd), http.StatusBadRequest)
		return
	}

	// Redis key per player per match
	redisKey := fmt.Sprintf("match:%d:player:%d:stats", MatchStat.MatchID, MatchStat.PlayerID)

	if hasReachedFoulLimit(redisKey) {
		// no further stat should be inserted as player should be out
		http.Error(w, "Player has 6 fouls, ignoring this stat", http.StatusForbidden)
		return
	}

	statJSON, err := json.Marshal(map[string]string{
		"minute": MatchStat.Minute,
		"stat":   MatchStat.Stat,
	})
	if err != nil {
		http.Error(w, "Failed to encode stat", http.StatusInternalServerError)
		return
	}

	if err := db.Redis.RPush(db.Ctx, redisKey, statJSON).Err(); err != nil {
		http.Error(w, "Failed to save stat to Redis", http.StatusInternalServerError)
		return
	}

	log.Printf("AddMatchStat: RedisKey: %s, Data: %s", redisKey, string(statJSON))
	w.WriteHeader(http.StatusOK)
}

func hasReachedFoulLimit(redisKey string) bool {
	stats, err := db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
	if err != nil {
		return false // fail open
	}

	foulCount := 0
	for _, item := range stats {
		var record map[string]string
		if err := json.Unmarshal([]byte(item), &record); err == nil {
			if record["stat"] == "fouls" {
				foulCount++
			}
		}
	}

	return foulCount >= 6
}

func GetPlayerMatchStat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, _ := strconv.Atoi(vars["matchId"])
	playerID, _ := strconv.Atoi(vars["playerId"])

	redisKey := fmt.Sprintf("match:%d:player:%d:stats", matchID, playerID)

	stats, err := db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
	if err != nil || len(stats) == 0 {
		http.Error(w, "No stats found for player", http.StatusNotFound)
		return
	}

	// Sort by minute as float
	slices.SortFunc(stats, func(a, b string) int {
		var am, bm map[string]string

		if err := json.Unmarshal([]byte(a), &am); err != nil {
			return 0
		}
		if err := json.Unmarshal([]byte(b), &bm); err != nil {
			return 0
		}

		amin, _ := strconv.ParseFloat(am["minute"], 64)
		bmin, _ := strconv.ParseFloat(bm["minute"], 64)

		if amin < bmin {
			return -1
		}
		if amin > bmin {
			return 1
		}
		return 0
	})

	query := r.URL.RawQuery
	requestedStats := make(map[string]bool)
	if query != "" {
		for _, stat := range strings.Split(query, ",") {
			stat = strings.ToLower(strings.TrimSpace(stat))
			if !slices.Contains(validStatsToFetch, stat) {
				http.Error(w, fmt.Sprintf("Invalid stat type. Available stats to fetch are: %v", validStatsToFetch), http.StatusBadRequest)
				return
			}
			requestedStats[stat] = true
		}
	}

	// Calculate sums
	statSums := make(map[string]interface{}) // Use interface{} to allow different types (int and float64)
	var parsedStats []map[string]string

	for _, stat := range stats {
		var record map[string]string
		if err := json.Unmarshal([]byte(stat), &record); err != nil {
			continue
		}
		parsedStats = append(parsedStats, record)
	}

	for _, record := range parsedStats {
		stat := record["stat"]

		if requestedStats[stat] {
			if _, exists := statSums[stat]; !exists {
				statSums[stat] = 0
			}
			statSums[stat] = statSums[stat].(int) + 1
		}

		// Points calculation (1pt, 2pt, 3pt)
		if requestedStats["points"] {
			if val, ok := pointValues[stat]; ok {
				// Initialize statSums["points"] if not already present
				if _, exists := statSums["points"]; !exists {
					statSums["points"] = 0 // Initialize with 0
				}
				// Ensure statSums["points"] is an int, and add the value
				statSums["points"] = statSums["points"].(int) + val
			}
		}
	}

	// Special case: calculate minutes based on "in" and "out" events
	if requestedStats["minutes"] {
		var totalMinutes string
		var inTime string

		for _, entry := range parsedStats {
			statName, ok := entry["stat"]
			if !ok {
				continue
			}

			minStr, ok := entry["minute"]
			if !ok {
				continue
			}

			switch statName {
			case "in":
				inTime = minStr
			case "out":
				if inTime != "" {
					totalMinutes = minutesAdd(totalMinutes, minuteDiff(inTime, minStr))
					inTime = ""
				}
			}
		}

		statSums["minutes"] = totalMinutes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statSums)
}

func minutesAdd(time1, time2 string) string {
	if time1 == "" {
		time1 = "00.00" // start of match
	}
	min1, sec1 := parseMinute(time1)
	min2, sec2 := parseMinute(time2)

	totalSec1 := min1*60 + sec1
	totalSec2 := min2*60 + sec2

	totalSec := totalSec1 + totalSec2

	resultMin := totalSec / 60
	resultSec := totalSec % 60

	return fmt.Sprintf("%02d.%02d", resultMin, resultSec)
}

func minuteDiff(fromStr, toStr string) string {
	fromMin, fromSec := parseMinute(fromStr)
	toMin, toSec := parseMinute(toStr)

	fromTotalSec := fromMin*60 + fromSec
	toTotalSec := toMin*60 + toSec

	diffSec := toTotalSec - fromTotalSec
	min := diffSec / 60
	sec := diffSec % 60

	return fmt.Sprintf("%02d.%02d", min, sec)
}

func parseMinute(minuteStr string) (int, int) {
	parts := strings.Split(minuteStr, ".")
	minutes, _ := strconv.Atoi(parts[0])
	seconds, _ := strconv.Atoi(parts[1])
	return minutes, seconds
}

func isValidMinuteValue(minute string) bool {
	val, err := strconv.ParseFloat(minute, 64)
	if err != nil {
		return false
	}
	return val >= 0.0 && val <= 48.0
}
