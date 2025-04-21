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

var validStats = []string{
	"rebounds", "assists", "steals", "blocks", "turnovers",
	"fouls", "minutes", "points", "1pt", "2pt", "3pt",
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

	if !slices.Contains(validStats, MatchStat.Stat) {
		http.Error(w, fmt.Sprintf("Invalid stat type. Available stats are: %v", validStats), http.StatusBadRequest)
		return
	}

	// Redis key per player per match
	redisKey := fmt.Sprintf("match:%d:player:%d:stats", MatchStat.MatchID, MatchStat.PlayerID)

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

	query := r.URL.RawQuery
	requestedStats := make(map[string]bool)
	if query != "" {
		for _, stat := range strings.Split(query, ",") {
			requestedStats[strings.ToLower(strings.TrimSpace(stat))] = true
		}
	}

	// Calculate sums
	statSums := make(map[string]int)
	for _, stat := range stats {
		var record map[string]string
		if err := json.Unmarshal([]byte(stat), &record); err != nil {
			continue
		}

		stat := record["stat"]

		// Calculate and return this stat in one of the cases:
		// 1. no filter of stat in query string (len(requestedStats) == 0)
		// 2. stat is requested in filter (requestedStats[stat])
		// 3. reqested for points (and stat is 1pt or 2pt or 3pt)
		if len(requestedStats) == 0 || requestedStats[stat] {
			statSums[stat]++
		}
		if val, ok := pointValues[stat]; ok && (len(requestedStats) == 0 || requestedStats["points"]) {
			statSums["points"] += val
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statSums)
}

func minuteDiff(fromStr, toStr string) float64 {
	fromMin, fromSec := parseMinute(fromStr)
	toMin, toSec := parseMinute(toStr)

	fromTotalSec := fromMin*60 + fromSec
	toTotalSec := toMin*60 + toSec

	diffSec := toTotalSec - fromTotalSec
	min := diffSec / 60
	sec := diffSec % 60

	return float64(min) + float64(sec)/100.0
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
