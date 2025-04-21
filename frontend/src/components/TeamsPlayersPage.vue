<template>
  <div id="app">
    <!-- Unassigned Players Section -->
    <div class="unassigned-players">
      <h2>Players Without a Team</h2>
      <div class="unassigned-list">
        <span
          v-for="player in getPlayersWithoutTeam()"
          :key="player.player_id"
          class="unassigned-player"
          draggable="true"
          @dragstart="draggedPlayer = player"
        >
          {{ player.first_name }} {{ player.last_name }}
        </span>
      </div>
    </div>

    <!-- Teams with Players -->
    <div class="teams-container">
      <div
        v-for="team in teams"
        :key="team.team_id"
        class="team-box"
        @dragover.prevent
        @drop="onDrop(team.team_id)"
      >
        <h3>{{ team.team_name }}</h3>
        <div class="players-list">
          <div
            v-for="player in getPlayersForTeam(team.team_id)"
            :key="player.player_id"
            class="player-box"
          >
          {{ player.first_name }} {{ player.last_name }} ({{ getStartDate(player.player_id, team.team_id) }})
        </div>
        </div>
      </div>
    </div>

    <!-- Date Picker Modal -->
    <div v-if="showDatePicker" class="modal-overlay">
      <div class="modal">
        <h3>Select Start Date</h3>
        <input type="date" v-model="selectedDate" />
        <div class="modal-buttons">
          <button @click="confirmAssignment">Confirm</button>
          <button @click="cancelAssignment">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  name: 'App',
  data() {
    return {
      teams: [],
      players: [],
      playerTeamHistory: [],
      draggedPlayer: null,
      dropTeamId: null,
      showDatePicker: false,
      selectedDate: ''
    };
  },
  mounted() {
    this.fetchData();
  },
  methods: {
    async fetchData() {
      try {
        const [teamsRes, playersRes, historyRes] = await Promise.all([
          axios.get('http://localhost:8080/api/teams'),
          axios.get('http://localhost:8080/api/players'),
          axios.get('http://localhost:8080/api/player_team_history')
        ]);

        this.teams = teamsRes.data;
        this.playerTeamHistory = historyRes.data;
        this.players = playersRes.data;
      } catch (error) {
        console.error('Error fetching data:', error);
      }
    },
    getPlayersForTeam(teamId) {
      const matchingPlayers = [];

      for (const history of this.playerTeamHistory) {
        if (
          history.teamId === teamId &&
          history.endDate &&
          history.endDate.Valid === false
        ) {
          for (const player of this.players) {
            const fullName = `${player.first_name} ${player.last_name}`;
            if (fullName === history.playerFullName) {
              matchingPlayers.push(player);
              break;
            }
          }
        }
      }
      return matchingPlayers;
    },
    getStartDate(playerId, teamId) {
      const player = this.players.find(p => p.player_id === playerId);
      if (!player) return 'N/A';

      const fullName = `${player.first_name} ${player.last_name}`;

      const history = this.playerTeamHistory.find(
        h =>
          h.playerFullName === fullName &&
          h.teamId === teamId &&
          h.endDate &&
          h.endDate.Valid === false
      );

      if (!history || !history.startDate) {
        console.warn('Missing or invalid startDate for player:', fullName, history);
        return 'N/A';
      }

      return history.startDate.split('T')[0];
    },
    getPlayersWithoutTeam() {
      const assigned = new Set();

      for (const history of this.playerTeamHistory) {
        if (history.endDate && history.endDate.Valid === false) {
          assigned.add(history.playerFullName);
        }
      }

      return this.players.filter(player => {
        const fullName = `${player.first_name} ${player.last_name}`;
        return !assigned.has(fullName);
      });
    },
    onDrop(teamId) {
      this.dropTeamId = teamId;
      this.showDatePicker = true;
    },
    async confirmAssignment() {
      if (!this.selectedDate || !this.draggedPlayer || !this.dropTeamId) return;

      const payload = {
        playerId: this.draggedPlayer.player_id,
        teamId: this.dropTeamId,
        startDate: this.selectedDate,
        endDate: null
      };

      try {
        await axios.post('http://localhost:8080/api/player_team_history', [payload]);
        this.showDatePicker = false;
        this.selectedDate = '';
        this.draggedPlayer = null;
        this.dropTeamId = null;
        await this.fetchData(); // Refresh
      } catch (err) {
        console.error('Error assigning player to team:', err);
      }
    },
    cancelAssignment() {
      this.showDatePicker = false;
      this.draggedPlayer = null;
      this.dropTeamId = null;
      this.selectedDate = '';
    }
  }
};
</script>

<style scoped>
#app {
  padding: 20px;
}

.unassigned-players {
  margin-bottom: 30px;
}

.unassigned-players h2 {
  margin-bottom: 10px;
  font-size: 1.2rem;
  color: #333;
}

.unassigned-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.unassigned-player {
  display: inline-block;
  background-color: #d1ecf1;
  color: #0c5460;
  padding: 6px 10px;
  border-radius: 15px;
  font-size: 0.9rem;
  white-space: nowrap;
  cursor: grab;
}

.teams-container {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  justify-content: flex-start;
}

.team-box {
  flex: 1 1 300px;
  max-width: 400px;
  border: 2px solid #aaa;
  border-radius: 10px;
  padding: 15px;
  background: #f8f8f8;
  box-shadow: 0 0 6px rgba(0, 0, 0, 0.1);
  min-height: 150px;
}

.players-list {
  margin-top: 10px;
}

.player-box {
  background: #e0e0e0;
  margin: 5px 0;
  padding: 5px;
  border-radius: 5px;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.4);
  display: flex;
  justify-content: center;
  align-items: center;
}

.modal {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 0 12px rgba(0, 0, 0, 0.3);
  text-align: center;
}

.modal-buttons {
  margin-top: 15px;
}

.modal-buttons button {
  margin: 0 10px;
  padding: 5px 15px;
}
</style>
