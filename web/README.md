# Frontend

React app for the Connect Four game. Pretty basic - just shows the game board and handles WebSocket connections.

## Setup

```bash
npm install
npm start
```

## What it does

- Shows 7x6 Connect Four board
- Connects to backend via WebSocket
- Handles player moves and game updates
- Shows leaderboard in a modal

## Main files

- `App.js` - Main game component
- `contexts/WebSocketContext.js` - WebSocket connection management
- `hooks/useWebSocketSimple.js` - Simplified WebSocket hook
- `components/Leaderboard.js` - Leaderboard component

The WebSocket hook was giving me issues with connection loops, so I made a simpler version that just works.