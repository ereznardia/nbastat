The statictics system has 2 modes, live mode and season mode.

Live page  - works with api calls that uses Redis to set and get stats data from it.

Season page  - works with api calls that uses Postgres to get stats data from it.

I have created a full UI (vuejs) infra, that works with the API (written in Go).

The UI was an extra effort I took to simplify the understanding of usage of the Go API.

The UI also allows to build the league with teams and players, so we can start a match and start working with live.go APIs (redis)

When we click end match, the match is synced into Postgres table.

Steps to work with the system:  (steps 1-5 are needed before we can actually work with the API as we need some data that the API requests for)

1. Run `docker compose up`

2. Then we need to populate some data into teams and players :
   
    - [POST] http://localhost:8080/api/players
    - take body from 'players.txt' file in the root folder
  
    - [POST] http://localhost:8080/api/teams
    - take body from 'teams.txt' file in the root folder
  
    - refresh the page and you should see the players and the teams.
  
3. To be able to start a match you should drag at least 5 players to 2 different teams.
   Drag player into a team and select any signing date in the team.

4. Select those 2 teams (they should be highlighted in blue) and click 'Enter'. In the popup select the match date. 
   The match date is meaningful as later in the season stats page it will fetch all matches that played in the season year.
   After selecting match data, it creates a new match in the 'matches' table.

5. Go to 'Matches' page, in the created match choose 5 opening players from both teams and click 'Start Match'.

6. Now we can start testing the API by clicking a player and simulate a stat for the player

   - stat (a list of stats rebounds/assists/fouls/...)
   - minute (between 0 to 48)
   - second (between 0 to 59)

  the API of course protects any invalid input.



Deployment to AWS

One option is to 
 1. Launch an EC2 instance
 2. Install Docker and Docker Compose
 3. I would setup some CI/CD pipeline to deploy code
 4. Write a docker-compose.yml with Go backend, Postgres, and Redis.
 5. docker-compose up -d
 6. Set up security groups to allow traffic on your app’s port




Tech Stack Decision:

Live Stats with Redis
I chose Redis to handle live game statistics due to its high performance and ability to efficiently handle multiple simultaneous updates. Each player maintains their own key in the format:
match:<matchId>:team:<teamId>:player:<playerId>,
which gets updated with a list of stat events throughout the match.

Persistent Storage with PostgreSQL
PostgreSQL is a solid choice for storing historical data. With approximately 82 games per season and between 200 to 400 stat events per team, it easily handles the required scale and ensures data integrity over time.


 

