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

func StartMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchIDStr, ok := vars["matchId"]
	if !ok {
		http.Error(w, "matchId is required in URL", http.StatusBadRequest)
		return
	}

	matchID, err := strconv.Atoi(matchIDStr)
	if err != nil {
		http.Error(w, "Invalid matchId format", http.StatusBadRequest)
		return
	}

	var teamPlayers map[int][]int
	if err := json.NewDecoder(r.Body).Decode(&teamPlayers); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(teamPlayers) != 2 {
		http.Error(w, "2 teams must be provided", http.StatusBadRequest)
		return
	}

	// Validate match exists and get home/away teams
	var homeTeamID, awayTeamID int
	err = db.PG.QueryRow(`
		SELECT home_team, away_team FROM matches WHERE match_id = $1
	`, matchID).Scan(&homeTeamID, &awayTeamID)
	if err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	for teamID, players := range teamPlayers {
		if teamID != homeTeamID && teamID != awayTeamID {
			http.Error(w, fmt.Sprintf("Team %d is not part of match %d", teamID, matchID), http.StatusBadRequest)
			return
		}

		if len(players) != 5 {
			http.Error(w, fmt.Sprintf("Team %d must have exactly 5 players", teamID), http.StatusBadRequest)
			return
		}

		// Fetch valid player IDs with no end_date
		rows, err := db.PG.Query(`
			SELECT player_id FROM player_team_history
			WHERE team_id = $1 AND end_date IS NULL
		`, teamID)
		if err != nil {
			http.Error(w, "Failed to query player-team history", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		validPlayers := make(map[int]bool)
		for rows.Next() {
			var pid int
			if err := rows.Scan(&pid); err == nil {
				validPlayers[pid] = true
			}
		}

		for _, playerID := range players {
			if !validPlayers[playerID] {
				http.Error(w, fmt.Sprintf("Player %d is not currently on team %d", playerID, teamID), http.StatusBadRequest)
				return
			}

			redisKey := fmt.Sprintf("match:%d:player:%d:stats", matchID, playerID)

			existingStats, err := db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
			if err != nil {
				http.Error(w, "Failed to read stats from Redis", http.StatusInternalServerError)
				return
			}

			skip := false
			for _, item := range existingStats {
				var record map[string]string
				if err := json.Unmarshal([]byte(item), &record); err == nil {
					if record["minute"] == "00.00" && record["stat"] == "in" {
						skip = true
						break
					}
				}
			}

			if skip {
				continue
			}

			statJSON, err := json.Marshal(map[string]string{
				"minute": "00.00",
				"stat":   "in",
			})
			if err != nil {
				http.Error(w, "Failed to encode stat", http.StatusInternalServerError)
				return
			}

			if err := db.Redis.RPush(db.Ctx, redisKey, statJSON).Err(); err != nil {
				http.Error(w, "Failed to save stat to Redis", http.StatusInternalServerError)
				return
			}
		}
	}

	log.Printf("StartMatch: Match %d started with teams: %+v", matchID, teamPlayers)
	w.WriteHeader(http.StatusOK)
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

	stats, err := db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
	if err != nil {
		return
	}

	if MatchStat.Stat != "in" {
		// when player is out of game, the only stat can be recorded of him is "in"
		if err := validatePlayerInPlay(stats); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if hasReachedFoulLimit(stats) {
		// no further stat should be inserted as player should be out
		http.Error(w, "Player reached 6 fouls, ignoring this stat", http.StatusForbidden)
		return
	}

	if MatchStat.Stat == "in" || MatchStat.Stat == "out" {
		if err := validateInOutSequence(stats, MatchStat.Stat); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

	w.WriteHeader(http.StatusOK)
}

func GetMatchStats(w http.ResponseWriter, r *http.Request) {
	keys, err := db.Redis.Keys(db.Ctx, "match:*:player:*:stats").Result()
	if err != nil {
		http.Error(w, "Failed to fetch match keys", http.StatusInternalServerError)
		return
	}

	matchSet := make(map[string]bool)

	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) >= 2 {
			matchSet[parts[1]] = true
		}
	}

	var matches []int
	for matchId := range matchSet {
		id, err := strconv.Atoi(matchId)
		if err == nil {
			matches = append(matches, id)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

func GetMatchStat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchId := vars["matchId"]

	pattern := "match:" + matchId + ":player:*:stats"
	keys, err := db.Redis.Keys(db.Ctx, pattern).Result()
	if err != nil {
		http.Error(w, "Failed to fetch keys", http.StatusInternalServerError)
		return
	}

	type Stat struct {
		Minute string `json:"minute"`
		Stat   string `json:"stat"`
	}

	result := make(map[string][]Stat)

	for _, key := range keys {
		data, err := db.Redis.LRange(db.Ctx, key, 0, -1).Result()
		if err != nil {
			continue
		}

		var stats []Stat
		for _, jsonStr := range data {
			var stat Stat
			if err := json.Unmarshal([]byte(jsonStr), &stat); err == nil {
				stats = append(stats, stat)
			}
		}

		// Extract player ID from key: match:3:player:101:stats → "101"
		parts := strings.Split(key, ":")
		if len(parts) >= 4 {
			playerID := parts[3]
			result[playerID] = stats
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func validatePlayerInPlay(stats []string) error {
	stats = sortStatsByMinute(stats)

	for i := len(stats) - 1; i >= 0; i-- {
		item := stats[i]

		var record map[string]string
		if err := json.Unmarshal([]byte(item), &record); err == nil {
			if record["stat"] == "in" {
				return nil
			} else if record["stat"] == "out" {
				return fmt.Errorf("player is out, can't add stat")
			}
		}
	}

	return fmt.Errorf("player is out, can't add stat")
}

func hasReachedFoulLimit(stats []string) bool {
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

func validateInOutSequence(stats []string, newStat string) error {
	stats = sortStatsByMinute(stats)

	var lastAction string

	for _, stat := range stats {
		var record map[string]string
		if err := json.Unmarshal([]byte(stat), &record); err != nil {
			continue
		}

		stat := record["stat"]
		if stat == "in" {
			lastAction = "in"
		} else if stat == "out" {
			lastAction = "out"
		}
	}

	if lastAction == "" && newStat == "out" {
		return fmt.Errorf("player can't go out before in")
	}

	if lastAction != newStat {
		return nil
	}
	return fmt.Errorf("invalid sequence of 'in' - 'out'")
}

func GetPlayerMatchStat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, _ := strconv.Atoi(vars["matchId"])
	playerID, _ := strconv.Atoi(vars["playerId"])

	redisKey := fmt.Sprintf("match:%d:player:%d:stats", matchID, playerID)

	// Get recorded stats from redis
	// and sort by minute (just in case it is not sorted)
	stats, err := db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
	if err != nil || len(stats) == 0 {
		http.Error(w, "No stats found for player", http.StatusNotFound)
		return
	}
	stats = sortStatsByMinute(stats)

	// Create this dictionary that maps stat to true/false
	// e.g if the api has query value like
	// "?rebounds,assists"
	// then:
	// requestedStats["rebounds"]=true and requestedStats["assists"]=true
	requestedStats, err := parseRequestedStats(r.URL.RawQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v. Available stats to fetch are: %v", err, validStatsToFetch), http.StatusBadRequest)
		return
	}

	// Calculate sums
	statSums := make(map[string]interface{}) // Use interface{} to allow different types (int for assists etc and float64 for minutes)
	var parsedStats []map[string]string

	// stats is array of strings representation of jsons, meed the parse it into 'parsedStats' so we can loop the stats
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
		var inTimes []string
		var totalMinutes string

		for i, entry := range parsedStats {
			stat := entry["stat"]
			minuteStr := entry["minute"]

			switch stat {
			case "in":
				inTimes = append(inTimes, minuteStr)
			case "out":
				if len(inTimes) > 0 {
					lastIn := inTimes[len(inTimes)-1]
					inTimes = inTimes[:len(inTimes)-1] // pop
					totalMinutes = minutesAdd(totalMinutes, minuteDiff(lastIn, minuteStr))
				}
			}

			// final check.. there will be no more iterations, and we have "ongoing" player, so we
			// will use the last stat's recorded minutes as the current "out"
			if i == len(parsedStats)-1 && len(inTimes) > 0 {
				lastMinute := minuteStr
				for _, pendingIn := range inTimes {
					totalMinutes = minutesAdd(totalMinutes, minuteDiff(pendingIn, lastMinute))
				}
			}
		}

		statSums["minutes"] = totalMinutes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statSums)
}

func parseRequestedStats(query string) (map[string]bool, error) {
	requestedStats := make(map[string]bool)

	if query == "" {
		// No query provided, return all valid stats
		for _, stat := range validStatsToFetch {
			requestedStats[stat] = true
		}
		return requestedStats, nil
	} else {
		for _, stat := range strings.Split(query, ",") {
			stat = strings.ToLower(strings.TrimSpace(stat))
			if !slices.Contains(validStatsToFetch, stat) {
				return nil, fmt.Errorf("invalid stat type: %s", stat)
			}
			requestedStats[stat] = true
		}
	}

	return requestedStats, nil
}

func ClearAllMatchStats() error {
	var cursor uint64
	match := "match:*:player:*:stats"

	for {
		keys, nextCursor, err := db.Redis.Scan(db.Ctx, cursor, match, 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %v", err)
		}

		if len(keys) > 0 {
			if err := db.Redis.Del(db.Ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func sortStatsByMinute(stats []string) []string {
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

	return stats
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
