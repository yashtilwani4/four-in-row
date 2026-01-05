import React, { useState, useEffect } from 'react';
import { WebSocketProvider, useWebSocketContext } from './contexts/WebSocketContext';
import Leaderboard from './components/Leaderboard';
import './App.css';

// Main App Component (wrapped with WebSocket context)
function AppContent() {
  const {
    // State
    connectionState,
    gameState,
    currentGame,
    playerId,
    playerName,
    message,
    lastError,
    isLoading,
    stats,
    
    // Connection state
    isConnected,
    isConnecting,
    isReconnecting,
    reconnectAttempts,
    
    // Actions
    setPlayerName,
    joinQueue,
    leaveQueue,
    makeMove,
    playAgain,
    clearError,
    
    // Constants
    GAME_STATES,
    WS_STATES
  } = useWebSocketContext();

  const [showLeaderboard, setShowLeaderboard] = useState(false);
  const [inputPlayerName, setInputPlayerName] = useState('');
  const [leaderboardRefreshTrigger, setLeaderboardRefreshTrigger] = useState(0);

  // Trigger leaderboard refresh when game finishes
  useEffect(() => {
    if (gameState === GAME_STATES.FINISHED) {
      // Delay refresh to allow backend to process the game result
      setTimeout(() => {
        setLeaderboardRefreshTrigger(prev => prev + 1);
      }, 1000);
    }
  }, [gameState, GAME_STATES.FINISHED]);

  // Handle join queue
  const handleJoinQueue = () => {
    if (!inputPlayerName.trim()) return;
    
    setPlayerName(inputPlayerName.trim());
    joinQueue(inputPlayerName.trim());
  };

  // Handle make move
  const handleMakeMove = (column) => {
    makeMove(column);
  };

  // Handle show leaderboard
  const handleShowLeaderboard = () => {
    setShowLeaderboard(true);
  };

  // Handle close leaderboard
  const handleCloseLeaderboard = () => {
    setShowLeaderboard(false);
  };

  // Render game board
  const renderBoard = () => {
    if (!currentGame || !currentGame.board) return null;

    const board = currentGame.board;
    const rows = [];

    // Add column headers for easier clicking
    const headers = [];
    for (let col = 0; col < 7; col++) {
      const isColumnFull = board[0][col] !== 0;
      headers.push(
        <div 
          key={`header-${col}`} 
          className={`column-header ${isColumnFull ? 'full' : ''}`}
          onClick={() => !isColumnFull && handleMakeMove(col)}
        >
          {col + 1}
        </div>
      );
    }
    rows.push(<div key="headers" className="column-headers">{headers}</div>);

    // Add board rows
    for (let row = 0; row < 6; row++) {
      const cells = [];
      for (let col = 0; col < 7; col++) {
        const cellValue = board[row][col];
        let cellClass = 'cell';
        
        if (cellValue === 1) cellClass += ' red';
        else if (cellValue === 2) cellClass += ' yellow';
        
        const isColumnFull = board[0][col] !== 0;
        
        cells.push(
          <div 
            key={`${row}-${col}`} 
            className={cellClass}
            onClick={() => !isColumnFull && handleMakeMove(col)}
          />
        );
      }
      rows.push(<div key={row} className="row">{cells}</div>);
    }

    return <div className="board">{rows}</div>;
  };

  // Get current player turn display
  const getCurrentPlayerTurn = () => {
    if (!currentGame || !playerId) return '';
    
    const myPlayer = currentGame.players.find(p => p.id === playerId);
    const currentPlayerNumber = currentGame.current_turn_number;
    const isMyTurn = myPlayer && myPlayer.number === currentPlayerNumber;
    const turnColor = currentPlayerNumber === 1 ? 'Red' : 'Yellow';
    const currentPlayer = currentGame.players.find(p => p.number === currentPlayerNumber);
    
    return isMyTurn ? `🎯 Your turn (${turnColor})` : `⏳ ${currentPlayer?.name}'s turn (${turnColor})`;
  };

  // Get connection status display
  const getConnectionStatusDisplay = () => {
    if (isReconnecting) {
      return `🟡 Reconnecting... (${reconnectAttempts}/10)`;
    }
    
    switch (connectionState) {
      case WS_STATES.CONNECTED:
        return '🟢 Connected';
      case WS_STATES.CONNECTING:
        return '🟡 Connecting...';
      case WS_STATES.DISCONNECTED:
        return '🔴 Disconnected';
      case WS_STATES.ERROR:
        return '🔴 Connection Error';
      case WS_STATES.CLOSED:
        return '⚫ Closed';
      default:
        return '🔴 Unknown';
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>🔴🟡 Connect Four</h1>
        
        <div className="status-bar">
          <span className="connection-status">{getConnectionStatusDisplay()}</span>
          <button 
            className="leaderboard-button"
            onClick={handleShowLeaderboard}
          >
            🏆 Leaderboard
          </button>
        </div>
        
        {/* Development stats */}
        {process.env.NODE_ENV === 'development' && (
          <div className="dev-stats">
            <small>
              Sent: {stats.messagesSent} | Received: {stats.messagesReceived} | Games: {stats.gamesPlayed}
            </small>
          </div>
        )}
        
        {lastError && (
          <div className="error" onClick={clearError}>
            ❌ {lastError}
          </div>
        )}
        
        {message && (
          <div className="message">
            ℹ️ {message}
          </div>
        )}
        
        {gameState === GAME_STATES.MENU && (
          <div className="menu">
            <div className="input-group">
              <input
                type="text"
                placeholder="Enter your username"
                value={inputPlayerName}
                onChange={(e) => setInputPlayerName(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleJoinQueue()}
                maxLength={20}
                disabled={!isConnected}
              />
              <button 
                onClick={handleJoinQueue} 
                disabled={!isConnected || !inputPlayerName.trim() || isLoading}
              >
                {isLoading ? '⏳ Finding...' : '🎮 Find Game'}
              </button>
            </div>
            
            <div className="menu-info">
              <p>Enter your username and click "Find Game" to start playing!</p>
              <p>You'll be matched with another player or play against a bot.</p>
              {!isConnected && (
                <p className="connection-warning">
                  ⚠️ Waiting for server connection...
                </p>
              )}
            </div>
          </div>
        )}
        
        {gameState === GAME_STATES.QUEUE && (
          <div className="queue">
            <div className="loading-spinner">⏳</div>
            <p>Looking for opponent...</p>
            <p className="queue-info">You'll be matched with a player or bot within 10 seconds</p>
            <button onClick={leaveQueue} disabled={!isConnected}>
              ❌ Cancel
            </button>
          </div>
        )}
        
        {(gameState === GAME_STATES.PLAYING || gameState === GAME_STATES.FINISHED) && (
          <div className="game">
            <div className="game-info">
              {currentGame && currentGame.players && (
                <div className="players-info">
                  <div className="player player-1">
                    <span className="player-color red-disc"></span>
                    {currentGame.players.find(p => p.number === 1)?.name || 'Player 1'}
                    {currentGame.players.find(p => p.number === 1)?.is_bot && ' 🤖'}
                  </div>
                  <div className="vs">VS</div>
                  <div className="player player-2">
                    <span className="player-color yellow-disc"></span>
                    {currentGame.players.find(p => p.number === 2)?.name || 'Player 2'}
                    {currentGame.players.find(p => p.number === 2)?.is_bot && ' 🤖'}
                  </div>
                </div>
              )}
              
              {gameState === GAME_STATES.PLAYING && (
                <div className="turn-indicator">
                  {getCurrentPlayerTurn()}
                </div>
              )}
              
              {currentGame && (
                <div className="game-stats">
                  <small>
                    Move {currentGame.move_count || 0} | 
                    Duration: {Math.floor((currentGame.duration || 0) / 60)}:{String((currentGame.duration || 0) % 60).padStart(2, '0')}
                  </small>
                </div>
              )}
            </div>
            
            {renderBoard()}
            
            {gameState === GAME_STATES.FINISHED && (
              <div className="game-finished">
                <button onClick={playAgain}>🎮 Play Again</button>
                <button onClick={handleShowLeaderboard}>🏆 View Leaderboard</button>
              </div>
            )}
          </div>
        )}
        
        {gameState === GAME_STATES.RECONNECTING && (
          <div className="reconnecting">
            <div className="loading-spinner">🔄</div>
            <p>Reconnecting to game...</p>
            <button onClick={playAgain}>❌ Cancel</button>
          </div>
        )}
        
        <Leaderboard
          isVisible={showLeaderboard}
          onClose={handleCloseLeaderboard}
          refreshTrigger={leaderboardRefreshTrigger}
        />
      </header>
    </div>
  );
}

// Main App component with WebSocket provider
function App() {
  return (
    <WebSocketProvider>
      <AppContent />
    </WebSocketProvider>
  );
}

export default App;