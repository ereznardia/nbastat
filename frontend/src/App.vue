<template>
  <div id="app">
    <!-- Navigation Menu -->
    <header>
      <nav>
        <ul>
          <li><a href="#" @click.prevent="showTeamsPlayers">Teams & Players</a></li>
          <li><a href="#" @click.prevent="showMatches">Matches</a></li>
          <li><a href="#" @click.prevent="showStatistics">Statistics</a></li>
        </ul>
      </nav>
    </header>

    <div v-if="currentView === 'teams-players'">
      <TeamsPlayersPage />
    </div>

    <div v-if="currentView === 'matches'">
      <MatchesPage />
    </div>

    <div v-if="currentView === 'statistics'">
      <StatisticsPage />
    </div>

  </div>
</template>

<script>
import TeamsPlayersPage from './components/TeamsPlayersPage.vue';
import MatchesPage from './components/MatchesPage.vue';
import StatisticsPage from './components/StatisticsPage.vue';

export default {
  name: 'App',
  components: {
    TeamsPlayersPage,
    MatchesPage,
    StatisticsPage,
  },
  data() {
    return {
      currentView: 'teams-players', // Default view
    };
  },
  methods: {
    showTeamsPlayers() {
      this.currentView = 'teams-players'; // Switch to Teams & Players
    },
    showMatches() {
      this.currentView = 'matches'; // Switch to Matches
    },
    showStatistics() {
      this.currentView = 'statistics'; // Switch to Statistics
    },
  },
};
</script>

<style scoped>
/* Basic styling for the navigation */
header {
  background-color: #282c34;
  padding: 20px;
}

nav ul {
  list-style-type: none;
  display: flex;
  gap: 20px;
}

nav li {
  color: white;
}

nav a {
  color: white;
  text-decoration: none;
  font-weight: bold;
}

nav a:hover {
  text-decoration: underline;
}
</style>
