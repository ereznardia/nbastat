<template>
  <div id="app" @keydown.enter="handleEnterKey" tabindex="0">
    <!-- Unassigned Players Section -->
    <div class="unassigned-players">
      <h2>Players Without a Team</h2>
      <div
          class="unassigned-list"
          @dragover.prevent
          @drop="handleUnassignDrop"
        >
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
        :class="{ highlighted: isTeamSelected(team) }"
        @click="selectTeam(team)"
        @dragover.prevent
        @drop="onDrop(team.team_id)"
      >
        <h3>{{ team.team_name }}</h3>
        <div class="players-list">
          <div
            v-for="player in getPlayersForTeam(team.team_id)"
            :key="player.player_id"
            class="player-box"
            draggable="true"
            @dragstart="draggedPlayer = player; dragSourceTeamId = team.team_id"
          >
            {{ player.first_name }} {{ player.last_name }} ({{ getStartDate(player.player_id, team.team_id) }})
          </div>
        </div>
      </div>
    </div>

    <!-- Player-Team Assignment Date Modal -->
    <div v-if="showTeamPlayerHistoryDatePicker" class="modal-overlay">
      <div class="modal">
        <h3>Select Start Date</h3>
        <input type="date" v-model="selectedTeamPlayerHistoryDate" />
        <div class="modal-buttons">
          <button @click="confirmAssignment">Confirm</button>
          <button @click="cancelAssignment">Cancel</button>
        </div>
      </div>
    </div>

    <!-- Match Creation Date Modal -->
    <div v-if="showMatchDatePicker" class="modal-overlay">
      <div class="modal">
        <h3>Select Match Date</h3>
        <input type="date" v-model="selectedMatchDate" />
        <div class="modal-buttons">
          <button @click="confirmMatch">Confirm</button>
          <button @click="cancelMatch">Cancel</button>
        </div>
      </div>
    </div>
  </div>

  <div v-if="showUnassignDatePicker" class="modal-overlay">
  <div class="modal">
    <h3>Select End Date</h3>
    <input type="date" v-model="selectedUnassignDate" />
    <div class="modal-buttons">
      <button @click="confirmUnassign">Confirm</button>
      <button @click="cancelUnassign">Cancel</button>
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
      showTeamPlayerHistoryDatePicker: false,
      showMatchDatePicker: false,
      selectedTeamPlayerHistoryDate: '',
      selectedMatchDate: '',
      selectedTeams: [],
      dragSourceTeamId: null,
      showUnassignDatePicker: false,
      selectedUnassignDate: '',
      unassignPlayer: null,
      unassignTeamId: null,
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
      this.showTeamPlayerHistoryDatePicker = true;
    },
    selectTeam(team) {
      const index = this.selectedTeams.findIndex(t => t.team_id === team.team_id);
      if (index === -1) {
        this.selectedTeams.push(team);
      } else {
        this.selectedTeams.splice(index, 1);
      }
    },
    isTeamSelected(team) {
      return this.selectedTeams.some(t => t.team_id === team.team_id);
    },
    async confirmAssignment() {
      if (!this.selectedTeamPlayerHistoryDate || !this.draggedPlayer || !this.dropTeamId) return;

      const payload = {
        playerId: this.draggedPlayer.player_id,
        teamId: this.dropTeamId,
        startDate: this.selectedTeamPlayerHistoryDate,
        endDate: null
      };

      try {
        await axios.post('http://localhost:8080/api/player_team_history', [payload]);
        this.showTeamPlayerHistoryDatePicker = false;
        this.selectedTeamPlayerHistoryDate = '';
        this.draggedPlayer = null;
        this.dropTeamId = null;
        this.dragSourceTeamId = null;
        await this.fetchData();
      } catch (err) {
        console.error('Error assigning player to team:', err);
      }
    },
    async confirmUnassign() {
      if (!this.selectedUnassignDate || !this.unassignPlayer || !this.unassignTeamId) return;

      let history = null;
      for (const h of this.playerTeamHistory) {
        if (
          h.playerId === this.unassignPlayer.player_id &&
          h.teamId === this.unassignTeamId &&
          h.endDate?.Valid === false
        ) {
          history = h;
          break;
        }
      }

      if (!history) {
        console.warn('No open assignment found to unassign.');
        return;
      }

      try {
        await axios.post(`http://localhost:8080/api/leave_team`, {
          player_id: this.unassignPlayer.player_id,
          team_id: this.unassignTeamId,
          end_date: this.selectedUnassignDate
        });

        this.showUnassignDatePicker = false;
        this.selectedUnassignDate = '';
        this.unassignPlayer = null;
        this.unassignTeamId = null;

        await this.fetchData();
      } catch (err) {
        console.error('Error unassigning player:', err);
      }
    },
    cancelAssignment() {
      this.showTeamPlayerHistoryDatePicker = false;
      this.draggedPlayer = null;
      this.dropTeamId = null;
      this.selectedTeamPlayerHistoryDate = '';
      this.dragSourceTeamId = null;
    },
    cancelUnassign() {
      this.showUnassignDatePicker = false;
      this.selectedUnassignDate = '';
      this.unassignPlayer = null;
      this.unassignTeamId = null;
    },
    handleEnterKey() {
      if (this.selectedTeams.length === 2) {
        this.showMatchDatePicker = true;
      } else {
        alert('Please select exactly two teams before creating a match.');
      }
    },
    async confirmMatch() {
      if (!this.selectedMatchDate || this.selectedTeams.length !== 2) return;

      const payload = [
          {
          homeTeam: String(this.selectedTeams[0].team_id),
          awayTeam: String(this.selectedTeams[1].team_id),
          date: this.selectedMatchDate
          }
      ];
      

      try {
        await axios.post('http://localhost:8080/api/matches', payload);
        alert('Match created successfully!');
        this.showMatchDatePicker = false;
        this.selectedMatchDate = '';
        this.selectedTeams = [];
      } catch (error) {
        console.error('Error creating match:', error);
      }
    },
    cancelMatch() {
      this.showMatchDatePicker = false;
      this.selectedMatchDate = '';
    },
    handleUnassignDrop() {
      if (!this.draggedPlayer || !this.dragSourceTeamId) return;

      this.unassignPlayer = this.draggedPlayer;
      this.unassignTeamId = this.dragSourceTeamId;
      this.showUnassignDatePicker = true;

      this.draggedPlayer = null;
      this.dragSourceTeamId = null;
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
  cursor: pointer;
}

.team-box.highlighted {
  background-color: #cce5ff;
}

.players-list {
  margin-top: 10px;
}

.player-box {
  background: #e0e0e0;
  margin: 5px 0;
  padding: 5px;
  border-radius: 5px;
  cursor: pointer;
}

.player-box.playerHighlighted {
  background-color: #d4edda;
  border: 1px solid #28a745;
  color: #155724;
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
