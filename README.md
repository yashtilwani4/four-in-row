# Connect Four - Multiplayer Game

A real-time Connect Four game I built for my backend engineering assignment. Players can play against each other or against an AI bot through WebSockets.

## What it does

- Real-time multiplayer Connect Four (7x6 board)
- AI bot opponent if no human player joins within 10 seconds
- WebSocket communication for live updates
- Leaderboard to track wins/losses
- React frontend with a simple game interface

## Tech Stack

**Backend:**
- Go with Gorilla WebSocket
- PostgreSQL for storing game results
- Kafka for analytics (optional)

**Frontend:**
- React with WebSocket connection
- Basic CSS for the game board

## Quick Start

### Easy way (just the game):
```bash
# Start backend
go run cmd/server/main.go

# Start frontend (in another terminal)
cd web && npm start
```

Then open http://localhost:3000

### Full setup with database:
```bash
# Copy environment file
cp .env.example .env

# Start PostgreSQL (you'll need Docker)
docker-compose up -d postgres

# Set up database tables
go run setup_database.go

# Start the server
go run cmd/server/main.go

# Start frontend
cd web && npm start
```

## How it works

### Game Flow
1. Player enters username and joins queue
2. System tries to match with another player
3. If no match in 10 seconds, creates game with AI bot
4. Players take turns dropping pieces
5. First to get 4 in a row wins
6. Game result saved to database

### AI Bot Strategy
The bot isn't random - it follows these priorities:
1. **Win** - Takes winning move if available
2. **Block** - Blocks opponent's winning move  
3. **Center** - Prefers center columns (better positioning)
4. **Setup** - Creates opportunities for future wins

### WebSocket Messages
```javascript
// Join game queue
{
  "type": "join_queue",
  "payload": {"player_name": "Alice"}
}

// Make a move
{
  "type": "make_move", 
  "payload": {"game_id": "...", "column": 3}
}
```

## Project Structure

```
├── cmd/server/          # Main server entry point
├── internal/
│   ├── game/           # Game state management
│   ├── bot/            # AI bot logic
│   ├── handlers/       # HTTP/WebSocket handlers
│   ├── database/       # Database stuff
│   └── models/         # Data structures
├── web/                # React frontend
└── docker-compose.yml  # For running PostgreSQL
```

## API Endpoints

- `GET /api/leaderboard` - Get player rankings
- `WS /ws` - WebSocket for game communication
- `GET /health` - Health check

## Database Schema

Two main tables:
- `games` - Stores completed games with winner, duration, etc.
- `leaderboard` - View that calculates player stats

## Testing

I wrote a few test files to make sure things work:
```bash
# Test WebSocket connection
go run test_simple_websocket.go

# Test full game flow
go run test_complete_flow.go

# Test database connection
go run test_db_connection.go
```

## Known Issues / TODO

- Kafka analytics is set up but not required for basic functionality
- Could add more sophisticated bot difficulty levels
- Frontend could use better styling
- Need to add proper error handling in some places

## Environment Variables

```bash
# .env file
DATABASE_URL=postgres://username:password@localhost:5432/connect_four
KAFKA_BROKERS=localhost:9092  # optional
PORT=8080
```

## Assignment Requirements Met

✅ Real-time multiplayer game server  
✅ WebSocket communication  
✅ AI bot with 10-second timeout  
✅ Game state persistence  
✅ Simple React frontend  
✅ Leaderboard system  
✅ 30-second reconnection grace period  

The Kafka analytics part was a bonus feature I added to learn about event streaming.