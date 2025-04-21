import { createApp } from 'vue';  // Correct import for Vue 3
import App from './App.vue';
import { createRouter, createWebHistory } from 'vue-router';  // Correct import for Vue 3 Router
import TeamsPlayersPage from './components/TeamsPlayersPage.vue';
import MatchesPage from './components/MatchesPage.vue';

// Define your routes
const routes = [
  {
    path: '/',
    name: 'teams-players',
    component: TeamsPlayersPage
  },
  {
    path: '/matches',
    name: 'matches',
    component: MatchesPage
  }
];

// Create router instance with history mode
const router = createRouter({
  history: createWebHistory(), // Use Web History for routing
  routes, // define your routes
});

// Create the app instance
const app = createApp(App);
app.use(router); // Tell the app to use the router
app.mount('#app'); // Mount the app on the DOM element with id "app"
