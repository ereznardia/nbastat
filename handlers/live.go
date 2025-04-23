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

	// Check if match is already started
	startKey := fmt.Sprintf("match:%d:started", matchID)
	started, err := db.Redis.Get(db.Ctx, startKey).Result()
	if err == nil && started == "true" {
		http.Error(w, "Match already started", http.StatusBadRequest)
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
	var date string
	var homeTeamID, awayTeamID int
	err = db.PG.QueryRow(`
		SELECT date, home_team, away_team FROM matches WHERE match_id = $1
	`, matchID).Scan(&date, &homeTeamID, &awayTeamID)
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
		}

		if err := db.Redis.Set(db.Ctx, fmt.Sprintf("match:%d:date", matchID), date, 0).Err(); err != nil {
			http.Error(w, "Failed to mark match date", http.StatusInternalServerError)
			return
		}

		for _, playerID := range players {
			redisKey := fmt.Sprintf("match:%d:team:%d:player:%d:stats", matchID, teamID, playerID)

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

			playerTeamKey := fmt.Sprintf("match:%d:player:%d:team", matchID, playerID)
			if err := db.Redis.Set(db.Ctx, playerTeamKey, teamID, 0).Err(); err != nil {
				http.Error(w, "Failed to map player to team in Redis", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := db.Redis.Set(db.Ctx, startKey, "true", 0).Err(); err != nil {
		http.Error(w, "Failed to mark match as started", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func EndMatch(w http.ResponseWriter, r *http.Request) {
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

	// Get all player stat keys for the match
	pattern := fmt.Sprintf("match:%d:team:*:player:*", matchID)
	iter := db.Redis.Scan(db.Ctx, 0, pattern, 0).Iterator()
	for iter.Next(db.Ctx) {
		playerKey := iter.Val()

		endMatchJSON, _ := json.Marshal(map[string]string{
			"minute": "48.00",
			"stat":   "out",
		})

		if err := db.Redis.RPush(db.Ctx, playerKey, endMatchJSON).Err(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save stat to Redis for key %s", playerKey), http.StatusInternalServerError)
			return
		}
	}
	if err := iter.Err(); err != nil {
		http.Error(w, "Error scanning Redis keys", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Match ended and stats updated."))

	// Sync Match from Redis into Database
	syncMatch(matchID)
}

func syncMatch(matchID int) {
	redisKey := fmt.Sprintf("match:%d:team:*:player:*:stats", matchID)

	keys, err := db.Redis.Keys(db.Ctx, redisKey).Result()
	if err != nil {
		log.Printf("Failed to fetch keys for match %d: %v", matchID, err)
		return
	}

	matchDate, err := db.Redis.Get(db.Ctx, fmt.Sprintf("match:%d:date", matchID)).Result()
	if err != nil {
		log.Printf("Failed to read match date: %v", err)
		return
	}

	successfullySynced := true

	for _, key := range keys {
		stats, err := db.Redis.LRange(db.Ctx, key, 0, -1).Result()
		if err != nil {
			log.Printf("Failed to fetch stats for key %s: %v", key, err)
			continue
		}

		for _, stat := range stats {
			var statData map[string]interface{}
			if err := json.Unmarshal([]byte(stat), &statData); err != nil {
				log.Printf("Failed to unmarshal stat data: %v", err)
				successfullySynced = false
				continue
			}

			minute := statData["minute"].(string)
			statType := statData["stat"].(string)

			parts := strings.Split(key, ":")
			teamID := parts[3]
			playerID := parts[5]

			_, err := db.PG.Exec(`
				INSERT INTO matches_stats (match_id, team_id, player_id, minute, stat, match_date)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, matchID, teamID, playerID, minute, statType, matchDate)
			if err != nil {
				log.Printf("Failed to insert stat into database: %v", err)
				successfullySynced = false
				break
			}
		}
	}

	var homeTeamID, awayTeamID int
	err = db.PG.QueryRow(`
				SELECT home_team, away_team FROM matches WHERE match_id = $1
			`, matchID).Scan(&homeTeamID, &awayTeamID)
	if err != nil {
		log.Printf("Failed to query database: %v", err)
		return
	}
	homeTeamScoreSummary, err1 := GetStatsSummary(matchID, "team", homeTeamID, "")
	awayTeamScoreSummary, err2 := GetStatsSummary(matchID, "team", awayTeamID, "")

	homePoints, ok := homeTeamScoreSummary["points"].(int)
	if !ok {
		homePoints = 0
	}

	awayPoints, ok := awayTeamScoreSummary["points"].(int)
	if !ok {
		awayPoints = 0
	}

	if err1 != nil || err2 != nil {
		log.Printf("Failed to get team stat", err)
	} else {
		query := `
			UPDATE matches
			SET home_score = $1 , away_score = $2
			WHERE match_id = $3`
		_, err := db.PG.Exec(query, homePoints, awayPoints, matchID)
		if err != nil {
			return
		}
	}

	if successfullySynced {
		pattern := fmt.Sprintf("match:%d:*", matchID)

		keys, err := db.Redis.Keys(db.Ctx, pattern).Result()
		if err != nil {
			log.Printf("Failed to fetch keys with pattern %s: %v", pattern, err)
			return
		}

		if len(keys) == 0 {
			log.Printf("No Redis keys found for match %d", matchID)
			return
		}

		if err := db.Redis.Del(db.Ctx, keys...).Err(); err != nil {
			log.Printf("Failed to delete Redis keys for match %d: %v", matchID, err)
		} else {
			log.Printf("Successfully deleted Redis keys for match %d", matchID)
		}
	}
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
	teamIDStr, err := db.Redis.Get(db.Ctx, fmt.Sprintf("match:%d:player:%d:team", MatchStat.MatchID, MatchStat.PlayerID)).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	teamId, err := strconv.Atoi(teamIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	redisKey := fmt.Sprintf("match:%d:team:%d:player:%d:stats", MatchStat.MatchID, teamId, MatchStat.PlayerID)

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
		if MatchStat.Stat != "out" {
			http.Error(w, "Player reached 6 fouls, ignoring this stat (unless it is out stat)", http.StatusForbidden)
			return
		}
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

	// player is out
	if hasReachedFoulLimit(stats) {
		statJSON, _ := json.Marshal(map[string]string{
			"minute": MatchStat.Minute,
			"stat":   "out",
		})
		db.Redis.RPush(db.Ctx, redisKey, statJSON)
	}

	w.WriteHeader(http.StatusOK)
}

func GetMatchStats(w http.ResponseWriter, r *http.Request) {
	keys, err := db.Redis.Keys(db.Ctx, "match:*:team:*:player:*:stats").Result()
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

	if len(matches) == 0 {
		json.NewEncoder(w).Encode([]int{})
	} else {
		json.NewEncoder(w).Encode(matches)
	}
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

func GetMatchStat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, _ := strconv.Atoi(vars["matchId"])
	entity := vars["entity"]
	entityID, _ := strconv.Atoi(vars["entityId"])

	summary, err := GetStatsSummary(matchID, entity, entityID, r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func GetStatsSummary(matchID int, entity string, entityID int, rawQuery string) (map[string]interface{}, error) {
	var redisKey string
	var stats []string
	var err error

	if entity == "team" {
		pattern := fmt.Sprintf("match:%d:team:%d:player:*:stats", matchID, entityID)
		keys, err := db.Redis.Keys(db.Ctx, pattern).Result()
		if err != nil {
			return nil, fmt.Errorf("error fetching keys")
		}
		for _, key := range keys {
			vals, err := db.Redis.LRange(db.Ctx, key, 0, -1).Result()
			if err == nil {
				stats = append(stats, vals...)
			}
		}
	} else if entity == "player" {
		teamIDStr, err := db.Redis.Get(db.Ctx, fmt.Sprintf("match:%d:player:%d:team", matchID, entityID)).Result()
		if err != nil {
			return nil, fmt.Errorf("cannot fetch team ID: %v", err)
		}
		teamID, err := strconv.Atoi(teamIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid team ID: %v", err)
		}
		redisKey = fmt.Sprintf("match:%d:team:%d:player:%d:stats", matchID, teamID, entityID)
		stats, err = db.Redis.LRange(db.Ctx, redisKey, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("no stats found")
		}
	} else {
		return nil, fmt.Errorf("invalid entity type")
	}

	stats = sortStatsByMinute(stats)

	requestedStats, err := parseRequestedStats(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("%v. Available stats to fetch are: %v", err, validStatsToFetch)
	}

	statSums := make(map[string]interface{})
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
		if requestedStats["points"] {
			if val, ok := pointValues[stat]; ok {
				if _, exists := statSums["points"]; !exists {
					statSums["points"] = 0
				}
				statSums["points"] = statSums["points"].(int) + val
			}
		}
	}

	if requestedStats["minutes"] {
		var inTime string
		var totalMinutes string

		for i, entry := range parsedStats {
			stat := entry["stat"]
			minuteStr := entry["minute"]
			switch stat {
			case "in":
				inTime = minuteStr
			case "out":
				if inTime != "" {
					totalMinutes = minutesAdd(totalMinutes, minuteDiff(inTime, minuteStr))
					inTime = ""
				}
			}
			if i == len(parsedStats)-1 && inTime != "" {
				totalMinutes = minutesAdd(totalMinutes, minuteDiff(inTime, lastMatchRecordedMinute(matchID)))
			}
		}
		statSums["minutes"] = totalMinutes
		statSums["in"] = (inTime != "")
	}

	return statSums, nil
}

func lastMatchRecordedMinute(matchID int) string {
	pattern := fmt.Sprintf("match:%d:team:*:player:*:stats", matchID)
	keys, err := db.Redis.Keys(db.Ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return "0" // fallback default
	}

	var maxMinute float64 = 0

	for _, key := range keys {
		// Get the last stat from the player's list
		lastEntry, err := db.Redis.LRange(db.Ctx, key, -1, -1).Result()
		if err != nil || len(lastEntry) == 0 {
			continue
		}

		var record map[string]string
		if err := json.Unmarshal([]byte(lastEntry[0]), &record); err != nil {
			continue
		}

		minStr := record["minute"]
		minVal, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			continue
		}

		if minVal > maxMinute {
			maxMinute = minVal
		}
	}

	return fmt.Sprintf("%.2f", maxMinute)
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
