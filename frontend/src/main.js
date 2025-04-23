import { createApp } from 'vue';  // Correct import for Vue 3
import App from './App.vue';
import TeamsPlayersPage from './components/TeamsPlayersPage.vue';
import MatchesPage from './components/MatchesPage.vue';
import StatisticsPage from './components/StatisticsPage.vue';

// Create the app instance
const app = createApp(App);

// Register components globally (optional)
app.component('TeamsPlayersPage', TeamsPlayersPage);
app.component('MatchesPage', MatchesPage);
app.component('StatisticsPage', StatisticsPage);

app.mount('#app'); // Mount the app on the DOM element with id "app"
