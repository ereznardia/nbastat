<template>
  <div class="matches-container">
    <h1>Matches</h1>
    <div v-if="matches.length === 0" class="no-matches">
      <p>No matches created yet.</p>
    </div>
    <div class="matches-grid">
      <div v-for="match in matches" :key="match.match_id" class="match-card">
        <div class="match-header">
          <h3>Match #{{ match.match_id }}</h3>
        </div>
        <div class="match-details">
          <p><strong>Home Team:</strong> {{ getTeamName(match.home_team) }}</p>
          <p><strong>Away Team:</strong> {{ getTeamName(match.away_team) }}</p>
          <p><strong>Date:</strong> {{ formatDate(match.date) }}</p>

          <h4>{{ getTeamName(match.home_team) }} Players</h4>
          <div>
            <div 
              v-for="player in activePlayers[match.home_team] || []" 
              :key="player.id"
              @click="togglePlayerSelection(match.match_id, 'home', player.id)"
              :class="[
                'player',
                { 'selected-player': isPlayerSelected(match.match_id, 'home', player.id) },
                { 'disabled-player': startedMatchIds.includes(match.match_id) },
                { 'highlighted-player': playerStats[match.match_id]?.[match.home_team]?.[player.id]?.in }
              ]">
              <div v-on:click="handlePlayerClick(match.match_id, player.id)" 
                :class="[
                  { 'normal': !startedMatchIds.includes(match.match_id) },
                  { 'bold': startedMatchIds.includes(match.match_id) }
                ]">
              {{ player.fullName }}
            </div>
            <div v-html="getPlayerStatsHtml(match.match_id, match.home_team, player.id)"></div>
            </div>
          </div>

          <h4>{{ getTeamName(match.away_team) }} Players</h4>
          <div>
            <div 
              v-for="player in activePlayers[match.away_team] || []" 
              :key="player.id"
              @click="togglePlayerSelection(match.match_id, 'away', player.id)"
              :class="[
                'player',
                { 'selected-player': isPlayerSelected(match.match_id, 'away', player.id) },
                { 'disabled-player': startedMatchIds.includes(match.match_id) },
                { 'highlighted-player': playerStats[match.match_id]?.[match.away_team]?.[player.id]?.in }
              ]">
              <div v-on:click="handlePlayerClick(match.match_id, player.id)" 
                :class="[
                  { 'normal': !startedMatchIds.includes(match.match_id) },
                  { 'bold': startedMatchIds.includes(match.match_id) }
                ]">
              {{ player.fullName }}
            </div>
            <div v-html="getPlayerStatsHtml(match.match_id, match.away_team, player.id)"></div>
            </div>
          </div>

          <button
            v-if="!startedMatchIds.includes(match.match_id)"
            @click="startMatch(match.match_id)"
            class="start-match-btn">
            Start Match
          </button>

          <button
            v-if="startedMatchIds.includes(match.match_id)"
            @click="endMatch(match.match_id)"
            class="end-match-btn">
            End Match
          </button>

        </div>
      </div>
    </div>
    <div v-if="isPopupVisible" class="popup-overlay">
      <div class="popup-content">
        <h3>Add Player Stat</h3>
        <label for="stat">Select Stat Type:</label>
        <select v-model="selectedStat" id="stat">
          <option v-for="(stat, index) in statTypes" :key="index" :value="stat">{{ stat }}</option>
        </select>

        <!-- <label for="minute">Minute (mm.ss):</label>
        <input type="text" id="minute" v-model="minute" placeholder="MM.SS" /> -->

        <label for="minute">Minute:</label>
        <input
          type="number"
          id="minute"
          v-model.number="minute"
          min="0"
          max="48"
          placeholder="MM"
        />

        <label for="second">Second:</label>
        <input
          type="number"
          id="second"
          v-model.number="second"
          min="0"
          max="59"
          placeholder="SS"
        />

        <button @click="submitStat">Confirm</button>
        <button @click="closePopup">Cancel</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      matches: [],
      teams: [],
      activePlayers: {},
      selectedPlayers: {},
      startedMatchIds: [],
      playerStats: {},
      isPopupVisible: false,
      selectedStat: '',
      minute: 0,
      second: 0,
      matchId: null,
      playerId: null,
      statTypes: ['rebounds', 'assists', 'steals', 'blocks', 'turnovers', 'fouls', 'in', 'out', '1pt', '2pt', '3pt']
    };
  },
  async created() {
    await this.fetchStartedMatches();
    await this.fetchTeams();
    await this.fetchMatches();
    this.startPolling();
  },
  methods: {    
    // Fetch teams data from API
    async fetchTeams() {
      try {
        const response = await fetch('/api/teams');
        if (response.ok) {
          this.teams = await response.json();
        }
      } catch (error) {
        console.error('Error fetching teams:', error);
      }
    },
    // Fetch matches and associated players and stats
    async fetchMatches() {
      try {
        const response = await fetch('/api/matches');
        if (response.ok) {
          const data = await response.json();
          this.matches = data;

          // Fetch players and stats for each match
          for (const match of data) {
            await this.fetchActivePlayers(match.home_team);
            await this.fetchActivePlayers(match.away_team);

            if (this.startedMatchIds.includes(match.match_id)) {
              await this.fetchPlayerStats(match.match_id, match.home_team);
              await this.fetchPlayerStats(match.match_id, match.away_team);
            }
          }
        }
      } catch (error) {
        console.error('Error fetching matches:', error);
      }
    },
    // Fetch active players for a team
    async fetchActivePlayers(teamId) {
      if (this.activePlayers[teamId]) return;

      try {
        const response = await fetch(`/api/team_active_players/${teamId}`);
        if (response.ok) {
          const players = await response.json();
          this.activePlayers[teamId] = players;  // Reactive update
        }
      } catch (error) {
        console.error(`Error fetching players for team ${teamId}:`, error);
      }
    },
    // Fetch the list of started matches
    async fetchStartedMatches() {
      try {
        const response = await fetch('/api/match_stats');
        if (response.ok) {
          this.startedMatchIds = await response.json();
        }
      } catch (error) {
        console.error('Error fetching started matches:', error);
      }
    },
    // Fetch player stats for a specific match and team
    async fetchPlayerStats(matchId, teamId) {
      const players = this.activePlayers[teamId] || [];

      for (const player of players) {
        try {
          const response = await fetch(`/api/match_stat/${matchId}/${player.id}`);
          if (response.ok) {
            const stats = await response.json();

            // Store stats and "in" status
            this.playerStats[matchId] = {
              ...this.playerStats[matchId],
              [teamId]: {
                ...this.playerStats[matchId]?.[teamId],
                [player.id]: stats
              }
            };
          }
        } catch (err) {
          console.error(`Failed to fetch stats for match ${matchId}, player ${player.id}:`, err);
        }
      }
    },
    // Get the team name by team ID
    getTeamName(teamId) {
      const team = this.teams.find(t => t.team_id === teamId);
      return team ? team.team_name : 'Unknown Team';
    },
    // Format the date for display
    formatDate(date) {
      const matchDate = new Date(date);
      return matchDate.toLocaleDateString();
    },
    // Toggle player selection for a match
    togglePlayerSelection(matchId, team, playerId) {
      if (this.startedMatchIds.includes(matchId)) return;

      const teamSelection = this.selectedPlayers[matchId]?.[team] || [];
      const index = teamSelection.indexOf(playerId);

      if (index === -1) {
        this.selectedPlayers[matchId] = {
          ...this.selectedPlayers[matchId],
          [team]: [...teamSelection, playerId]
        };
      } else {
        teamSelection.splice(index, 1);
      }
    },
    // Check if a player is selected for a match
    isPlayerSelected(matchId, team, playerId) {
      return this.selectedPlayers[matchId]?.[team]?.includes(playerId);
    },
    // Start the match after player selection
    async startMatch(matchId) {
      const playersData = this.preparePlayersData(matchId);

      if (Object.keys(playersData).length === 0) {
        alert('Please select players for both teams');
        return;
      }

      try {
        const response = await fetch(`/api/start_match/${matchId}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(playersData),
        });

        if (response.ok) {
          this.startedMatchIds.push(matchId);
        } else {
          console.error('Failed to start match');
        }
      } catch (error) {
        console.error('Error starting match:', error);
      }
    },
    async endMatch(matchId) {
      try {
        const response = await fetch(`/api/end_match/${matchId}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          }
        });

        if (response.ok) {
          // this.startedMatchIds.push(matchId);
        } else {
          console.error('Failed to end match');
        }
      } catch (error) {
        console.error('Error ending match:', error);
      }
    },
    // Prepare player data for starting the match
    preparePlayersData(matchId) {
      const match = this.matches.find(m => m.match_id === matchId);
      if (!match) return {};

      const home = this.selectedPlayers[matchId]?.home || [];
      const away = this.selectedPlayers[matchId]?.away || [];

      const playersData = {};
      if (home.length) playersData[match.home_team] = home;
      if (away.length) playersData[match.away_team] = away;

      return playersData;
    },
    // Generate the HTML for player stats
    getPlayerStatsHtml(matchId, teamId, playerId) {
      const stats = this.playerStats[matchId]?.[teamId]?.[playerId];
      if (!stats) return '';

      let statsHtml = '';
      // Dynamically add only existing stats
      const statLabels = ['rebounds', 'assists', 'steals', 'blocks', 'turnovers', 'fouls', 'minutes', '1pt', '2pt', '3pt', 'points'];

      statLabels.forEach(stat => {
        if (stats[stat] !== undefined) {
          statsHtml += `<div><span>${stat}: </span> ${stats[stat]}</div>`;
        }
      });

      return statsHtml;
    },
    showAddStatPopup(matchId, playerId) {
      this.currentMatchId = matchId;
      this.currentPlayerId = playerId;
      this.showStatPopup = true;
      this.selectedStatType = 'points'; // Default stat type
      this.minute = ''; // Clear the minute input
      this.second = ''; // Clear the minute input
    },

    closeStatPopup() {
      this.showStatPopup = false;
    },
    
    formatTwoDigitsString(value) {
      return value < 10 ? `0${value}` : `${value}`;
    },
    handlePlayerClick(matchId, playerId) {
      if (this.startedMatchIds.includes(matchId)) {
        // Show the popup to add stats
        this.matchId = matchId;
        this.playerId = playerId;
        this.isPopupVisible = true;
      }
    },    
    closePopup() {
      this.isPopupVisible = false;
      this.selectedStat = '';
      this.minute = '';
    },
    
    async submitStat() {
      if (!this.selectedStat || !this.matchId || !this.playerId) {
        alert('Please complete all fields.');
        return;
      }

      const formattedMinute = this.formatTwoDigitsString(this.minute);
      const formattedSecond = this.formatTwoDigitsString(this.second);

      const statData = {
        matchId: this.matchId,
        playerId: this.playerId,
        minute: formattedMinute + "." + formattedSecond,
        stat: this.selectedStat
      };

      try {
        const response = await fetch('/api/match_stat', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(statData)
        });

        if (response.ok) {
          this.closePopup(); // Close the popup after submission
        } else {
          console.error('Error adding stat');
          alert('Failed to add stat. Please try again.');
        }
      } catch (error) {
        console.error('Error submitting stat:', error);
        alert('Error submitting stat');
      }
    },
    startPolling() {
      this.pollInterval = setInterval(() => {
        this.fetchMatches();
      }, 1000);
    },
    stopPolling() {
      clearInterval(this.pollInterval);
    }
  }
};
</script>

<style scoped>
.matches-container {
  padding: 20px;
  font-family: Arial, sans-serif;
  text-align: center;
}

.matches-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
  margin-top: 20px;
}

.match-card {
  background-color: #f4f4f4;
  border-radius: 8px;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  padding: 20px;
  text-align: left;
}

.match-header h3 {
  font-size: 1.2em;
  color: #333;
}

.match-details p {
  font-size: 1em;
  color: #555;
  margin: 5px 0;
}

h4 {
  font-size: 1.1em;
  margin-top: 10px;
  color: #333;
}

.player {
  padding: 5px;
  margin: 4px 0;
  cursor: pointer;
  border-radius: 4px;
}

.selected-player {
  background-color: #ede2a6;
}

.disabled-player {
  background-color: #ccc;
}

.start-match-btn {
  margin-top: 15px;
  padding: 10px;
  background-color: #4caf50;
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

.start-match-btn:hover {
  background-color: #45a049;
}

.end-match-btn {
  margin-top: 15px;
  padding: 10px;
  background-color: #e83008;
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

.end-match-btn:hover {
  background-color: #e23008;
}

.match-started-label {
  margin-top: 10px;
  font-weight: bold;
  color: #888;
}

.no-matches {
  font-size: 1.2em;
  color: #888;
  margin-top: 20px;
}


.stat-popup {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.popup-content {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  max-width: 400px;
  width: 100%;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.popup-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 15px;
}

.popup-actions button {
  padding: 8px 15px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

.popup-actions button:first-child {
  background-color: #4caf50;
  color: white;
}

.popup-actions button:first-child:hover {
  background-color: #45a049;
}

.popup-actions button:last-child {
  background-color: #ccc;
}

.popup-actions button:last-child:hover {
  background-color: #999;
}
.bold{
  background-color: #222;
  color: #ccc;
}
.popup-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.popup-content {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  width: 300px;
  text-align: center;
}

.popup-content label {
  display: block;
  margin-bottom: 8px;
}

.popup-content input,
.popup-content select {
  margin-bottom: 15px;
  padding: 8px;
  width: 100%;
  border-radius: 4px;
  border: 1px solid #ccc;
}

.popup-content button {
  margin-top: 10px;
  padding: 8px 16px;
  background-color: #4CAF50;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.popup-content button:hover {
  background-color: #45a049;
}

.popup-content button:last-child {
  background-color: #f44336;
}

.popup-content button:last-child:hover {
  background-color: #e53935;
}
.highlighted-player {
  background-color: #e0ffe0;
  border: 2px solid green;
}
</style>
