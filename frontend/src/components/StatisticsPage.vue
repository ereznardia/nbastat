<template>
  <div class="statistics-page">
    <div class="filters">
      <select v-model="selectedYear" @change="fetchStats">
        <option value="" disabled>Select Season</option>
        <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
      </select>

      <select v-model="selectedStat" @change="fetchStats">
        <option value="" disabled>Select Stat</option>
        <option v-for="stat in statsList" :key="stat" :value="stat">{{ stat }}</option>
      </select>
    </div>

    <div class="stats-container">
      <!-- Players List -->
      <div class="list-section">
        <h3>Players</h3>
        <div v-if="players.length">
          <div v-for="player in players" :key="player.player_id" class="item-row">
            <div>
              {{ player.first_name }} {{ player.last_name }}
              <span
                v-if="playerStats[player.player_id] !== undefined"
                class="stat-badge"
              >
                {{ playerStats[player.player_id].toFixed(2) }}
              </span>
            </div>
          </div>
        </div>
        <p v-else>Loading players...</p>
      </div>

      <!-- Teams List -->
      <div class="list-section">
        <h3>Teams</h3>
        <div v-if="teams.length">
          <div v-for="team in teams" :key="team.team_id" class="item-row">
            <div>
              {{ team.team_name }}
              <span
                v-if="teamStats[team.team_id] !== undefined"
                class="stat-badge"
              >
                {{ teamStats[team.team_id].toFixed(2) }}
              </span>
            </div>
          </div>
        </div>
        <p v-else>Loading teams...</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'StatisticsPage',
  data() {
    return {
      players: [],
      teams: [],
      playerStats: {},
      teamStats: {},
      years: Array.from({ length: 2025 - 2000 + 1 }, (_, i) => 2000 + i),
      statsList: [
        "rebounds", "assists", "steals", "blocks", "turnovers",
        "fouls", "minutes", "1pt", "2pt", "3pt", "points"
      ],
      selectedYear: "",
      selectedStat: "",
    };
  },
  mounted() {
    // Fetch players and teams on mount
    fetch('/api/players')
      .then(res => res.json())
      .then(data => { this.players = data; });

    fetch('/api/teams')
      .then(res => res.json())
      .then(data => { this.teams = data; });
  },
  methods: {
    async fetchStats() {
      if (!this.selectedYear || !this.selectedStat) return;

      // --- Player stats ---
      const playerFetches = this.players.map(async player => {
        const url = `/api/${this.selectedYear}/player/${player.player_id}/${this.selectedStat}`;
        try {
          const res = await fetch(url);
          const data = await res.json();
          this.playerStats[player.player_id] = data.average ?? 0;
        } catch (e) {
          this.playerStats[player.player_id] = 0;
        }
      });

      // --- Team stats ---
      const teamFetches = this.teams.map(async team => {
        const url = `/api/${this.selectedYear}/team/${team.team_id}/${this.selectedStat}`;
        try {
          const res = await fetch(url);
          const data = await res.json();
          this.teamStats[team.team_id] = data.average ?? 0;
        } catch (e) {
          this.teamStats[team.team_id] = 0;
        }
      });

      await Promise.all([...playerFetches, ...teamFetches]);

      // Sort both lists descending by stat
      this.players.sort((a, b) => (this.playerStats[b.player_id] ?? 0) - (this.playerStats[a.player_id] ?? 0));
      this.teams.sort((a, b) => (this.teamStats[b.team_id] ?? 0) - (this.teamStats[a.team_id] ?? 0));
    },
  }
};
</script>

<style scoped>
.statistics-page {
  padding: 20px;
}

.filters {
  margin-bottom: 20px;
}

select {
  padding: 8px;
  margin-right: 10px;
}

.stats-container {
  display: flex;
  gap: 40px;
}

.list-section {
  flex: 1;
}

.item-row {
  padding: 10px;
  border-bottom: 1px solid #ccc;
  font-size: 16px;
}

.stat-badge {
  background-color: #000077;
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 10px;
}
</style>
