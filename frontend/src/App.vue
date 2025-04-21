<template>
  <div id="app">
    <!-- Navigation Menu -->
    <header>
      <nav>
        <ul>
          <li><router-link to="/">Teams & Players</router-link></li>
          <li><router-link to="/matches">Matches</router-link></li>
        </ul>
      </nav>
    </header>
    
    <!-- This is where the routed components will be displayed -->
    <router-view></router-view>
  </div>
</template>

<script>
export default {
  name: 'App'
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
