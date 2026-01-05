import React, { createContext, useContext, useReducer, useEffect } from 'react';
import { useWebSocket, MESSAGE_TYPES, WS_STATES } from '../hooks/useWebSocket';

// Game states
export const GAME_STATES = {
  MENU: 'menu',
  QUEUE: 'queue',
  PLAYING: 'playing',
  FINISHED: 'finished',
  RECONNECTING: 'reconnecting'
};

// Initial state
const initialState = {
  // Connection state
  connectionState: WS_STATES.DISCONNECTED,
  lastError: null,
  
  // Game state
  gameState: GAME_STATES.MENU,
  currentGame: null,
  playerId: null,
  playerName: '',
  
  // UI state
  message: '',
  isLoading: false,
  
  // Statistics
  stats: {
    messagesSent: 0,
    messagesReceived: 0,
    gamesPlayed: 0,
    connectionTime: null
  }
};

// Action types
const ACTION_TYPES = {
  SET_CONNECTION_STATE: 'SET_CONNECTION_STATE',
  SET_GAME_STATE: 'SET_GAME_STATE',
  SET_PLAYER_NAME: 'SET_PLAYER_NAME',
  SET_CURRENT_GAME: 'SET_CURRENT_GAME',
  SET_PLAYER_ID: 'SET_PLAYER_ID',
  SET_MESSAGE: 'SET_MESSAGE',
  SET_ERROR: 'SET_ERROR',
  SET_LOADING: 'SET_LOADING',
  UPDATE_STATS: 'UPDATE_STATS',
  RESET_GAME: 'RESET_GAME'
};

// Reducer
const webSocketReducer = (state, action) => {
  switch (action.type) {
    case ACTION_TYPES.SET_CONNECTION_STATE:
      return {
        ...state,
        connectionState: action.payload
      };
      
    case ACTION_TYPES.SET_GAME_STATE:
      return {
        ...state,
        gameState: action.payload
      };
      
    case ACTION_TYPES.SET_PLAYER_NAME:
      return {
        ...state,
        playerName: action.payload
      };
      
    case ACTION_TYPES.SET_CURRENT_GAME:
      return {
        ...state,
        currentGame: action.payload
      };
      
    case ACTION_TYPES.SET_PLAYER_ID:
      return {
        ...state,
        playerId: action.payload
      };
      
    case ACTION_TYPES.SET_MESSAGE:
      return {
        ...state,
        message: action.payload,
        lastError: null
      };
      
    case ACTION_TYPES.SET_ERROR:
      return {
        ...state,
        lastError: action.payload,
        message: ''
      };
      
    case ACTION_TYPES.SET_LOADING:
      return {
        ...state,
        isLoading: action.payload
      };
      
    case ACTION_TYPES.UPDATE_STATS:
      return {
        ...state,
        stats: {
          ...state.stats,
          ...action.payload
        }
      };
      
    case ACTION_TYPES.RESET_GAME:
      return {
        ...state,
        gameState: GAME_STATES.MENU,
        currentGame: null,
        playerId: null,
        message: '',
        lastError: null,
        isLoading: false
      };
      
    default:
      return state;
  }
};

// Create context
const WebSocketContext = createContext();

// WebSocket Provider Component
export const WebSocketProvider = ({ children }) => {
  const [state, dispatch] = useReducer(webSocketReducer, initialState);
  
  // Initialize WebSocket hook
  const webSocket = useWebSocket();

  // Update connection state when WebSocket state changes
  useEffect(() => {
    dispatch({
      type: ACTION_TYPES.SET_CONNECTION_STATE,
      payload: webSocket.connectionState
    });
    
    if (webSocket.lastError) {
      dispatch({
        type: ACTION_TYPES.SET_ERROR,
        payload: webSocket.lastError
      });
    }
  }, [webSocket.connectionState, webSocket.lastError]);

  // Message handlers
  useEffect(() => {
    // Game found handler
    const unsubscribeGameFound = webSocket.onMessage(MESSAGE_TYPES.GAME_FOUND, (message) => {
      const { game, player_id } = message.payload;
      
      dispatch({ type: ACTION_TYPES.SET_CURRENT_GAME, payload: game });
      dispatch({ type: ACTION_TYPES.SET_PLAYER_ID, payload: player_id });
      dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.PLAYING });
      dispatch({ type: ACTION_TYPES.SET_LOADING, payload: false });
      
      // Determine player info
      const myPlayer = game.players.find(p => p.id === player_id);
      const opponent = game.players.find(p => p.id !== player_id);
      const myColor = myPlayer?.number === 1 ? 'Red' : 'Yellow';
      const opponentName = opponent?.name || 'Opponent';
      const botIndicator = opponent?.is_bot ? ' 🤖' : '';
      
      dispatch({
        type: ACTION_TYPES.SET_MESSAGE,
        payload: `🎮 Game found! You are ${myColor} vs ${opponentName}${botIndicator}`
      });
    });

    // Move result handler
    const unsubscribeMoveResult = webSocket.onMessage(MESSAGE_TYPES.MOVE_RESULT, (message) => {
      const { success, game_state, error, is_game_over, win_result } = message.payload;
      
      if (success) {
        dispatch({ type: ACTION_TYPES.SET_CURRENT_GAME, payload: game_state });
        
        if (is_game_over) {
          dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.FINISHED });
          
          if (win_result && win_result.has_winner) {
            const winnerPlayer = game_state.players.find(p => p.number === win_result.winner);
            const isWinner = winnerPlayer && winnerPlayer.id === state.playerId;
            const winnerColor = win_result.winner === 1 ? 'Red' : 'Yellow';
            const winType = win_result.win_type;
            
            dispatch({
              type: ACTION_TYPES.SET_MESSAGE,
              payload: isWinner 
                ? `🎉 You won with a ${winType} connection! (${winnerColor})`
                : `😞 You lost! ${winnerPlayer?.name} won with ${winType} (${winnerColor})`
            });
          } else {
            dispatch({
              type: ACTION_TYPES.SET_MESSAGE,
              payload: "🤝 It's a draw! Great game!"
            });
          }
          
          // Update games played stat
          dispatch({
            type: ACTION_TYPES.UPDATE_STATS,
            payload: { gamesPlayed: state.stats.gamesPlayed + 1 }
          });
        }
      } else {
        dispatch({ type: ACTION_TYPES.SET_ERROR, payload: error });
        // Clear error after 3 seconds
        setTimeout(() => {
          dispatch({ type: ACTION_TYPES.SET_ERROR, payload: null });
        }, 3000);
      }
    });

    // Game end handler
    const unsubscribeGameEnd = webSocket.onMessage(MESSAGE_TYPES.GAME_END, (message) => {
      const { game_state, winner, is_draw, reason } = message.payload;
      
      dispatch({ type: ACTION_TYPES.SET_CURRENT_GAME, payload: game_state });
      dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.FINISHED });
      
      if (is_draw) {
        dispatch({
          type: ACTION_TYPES.SET_MESSAGE,
          payload: "🤝 Game ended in a draw!"
        });
      } else if (winner) {
        const isWinner = winner.id === state.playerId;
        dispatch({
          type: ACTION_TYPES.SET_MESSAGE,
          payload: isWinner 
            ? `🎉 You won! Congratulations!`
            : `😞 ${winner.name} won! Better luck next time!`
        });
      }
      
      if (reason && reason !== 'game_completed') {
        dispatch({
          type: ACTION_TYPES.SET_MESSAGE,
          payload: `⚠️ Game ended: ${reason}`
        });
      }
    });

    // Player disconnected handler
    const unsubscribePlayerDisconnected = webSocket.onMessage(MESSAGE_TYPES.PLAYER_DISCONNECTED, (message) => {
      dispatch({
        type: ACTION_TYPES.SET_MESSAGE,
        payload: '⚠️ Opponent disconnected. Waiting for reconnection...'
      });
    });

    // Player reconnected handler
    const unsubscribePlayerReconnected = webSocket.onMessage(MESSAGE_TYPES.PLAYER_RECONNECTED, (message) => {
      dispatch({
        type: ACTION_TYPES.SET_MESSAGE,
        payload: '✅ Opponent reconnected. Game continues!'
      });
      
      // Clear message after 3 seconds
      setTimeout(() => {
        dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '' });
      }, 3000);
    });

    // Bot move handler
    const unsubscribeBotMove = webSocket.onMessage(MESSAGE_TYPES.BOT_MOVE, (message) => {
      const { game_state, reasoning, confidence } = message.payload;
      
      dispatch({ type: ACTION_TYPES.SET_CURRENT_GAME, payload: game_state });
      
      if (reasoning && process.env.NODE_ENV === 'development') {
        console.log(`🤖 Bot move: ${reasoning} (confidence: ${confidence}%)`);
      }
    });

    // Error handler
    const unsubscribeError = webSocket.onMessage(MESSAGE_TYPES.ERROR, (message) => {
      const errorMsg = message.payload?.message || message.payload?.error || 'Unknown error';
      dispatch({ type: ACTION_TYPES.SET_ERROR, payload: errorMsg });
      dispatch({ type: ACTION_TYPES.SET_LOADING, payload: false });
    });

    // Reconnect success handler
    const unsubscribeReconnectSuccess = webSocket.onMessage(MESSAGE_TYPES.RECONNECT_SUCCESS, (message) => {
      const { game_state, queued_messages } = message.payload;
      
      if (game_state) {
        dispatch({ type: ACTION_TYPES.SET_CURRENT_GAME, payload: game_state });
        dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.PLAYING });
      }
      
      dispatch({
        type: ACTION_TYPES.SET_MESSAGE,
        payload: `✅ Reconnected successfully! ${queued_messages ? `Received ${queued_messages} queued messages.` : ''}`
      });
      
      // Clear message after 3 seconds
      setTimeout(() => {
        dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '' });
      }, 3000);
    });

    // Return cleanup functions
    return () => {
      unsubscribeGameFound();
      unsubscribeMoveResult();
      unsubscribeGameEnd();
      unsubscribePlayerDisconnected();
      unsubscribePlayerReconnected();
      unsubscribeBotMove();
      unsubscribeError();
      unsubscribeReconnectSuccess();
    };
  }, [webSocket, state.playerId, state.stats.gamesPlayed]);

  // Action creators
  const actions = {
    setPlayerName: (name) => {
      dispatch({ type: ACTION_TYPES.SET_PLAYER_NAME, payload: name });
    },
    
    joinQueue: (playerName) => {
      if (!webSocket.isConnected) {
        dispatch({ type: ACTION_TYPES.SET_ERROR, payload: 'Not connected to server' });
        return false;
      }
      
      dispatch({ type: ACTION_TYPES.SET_PLAYER_NAME, payload: playerName });
      dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.QUEUE });
      dispatch({ type: ACTION_TYPES.SET_LOADING, payload: true });
      dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '🔍 Looking for opponent...' });
      
      return webSocket.joinQueue(playerName);
    },
    
    leaveQueue: () => {
      dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.MENU });
      dispatch({ type: ACTION_TYPES.SET_LOADING, payload: false });
      dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '' });
      
      return webSocket.leaveQueue();
    },
    
    makeMove: (column) => {
      if (!state.currentGame || !state.playerId) {
        dispatch({ type: ACTION_TYPES.SET_ERROR, payload: 'No active game' });
        return false;
      }
      
      // Validate turn
      const myPlayer = state.currentGame.players.find(p => p.id === state.playerId);
      if (!myPlayer || myPlayer.number !== state.currentGame.current_turn) {
        dispatch({ type: ACTION_TYPES.SET_ERROR, payload: "It's not your turn!" });
        setTimeout(() => {
          dispatch({ type: ACTION_TYPES.SET_ERROR, payload: null });
        }, 2000);
        return false;
      }
      
      // Validate column
      if (state.currentGame.board && state.currentGame.board[0][column] !== 0) {
        dispatch({ type: ACTION_TYPES.SET_ERROR, payload: "Column is full!" });
        setTimeout(() => {
          dispatch({ type: ACTION_TYPES.SET_ERROR, payload: null });
        }, 2000);
        return false;
      }
      
      return webSocket.makeMove(state.currentGame.id, column);
    },
    
    playAgain: () => {
      dispatch({ type: ACTION_TYPES.RESET_GAME });
    },
    
    reconnectToGame: (gameId, playerId, username) => {
      dispatch({ type: ACTION_TYPES.SET_GAME_STATE, payload: GAME_STATES.RECONNECTING });
      dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '🔄 Reconnecting to game...' });
      
      return webSocket.reconnectToGame(gameId, playerId, username);
    },
    
    clearError: () => {
      dispatch({ type: ACTION_TYPES.SET_ERROR, payload: null });
    },
    
    clearMessage: () => {
      dispatch({ type: ACTION_TYPES.SET_MESSAGE, payload: '' });
    }
  };

  // Context value
  const contextValue = {
    // State
    ...state,
    
    // WebSocket state
    isConnected: webSocket.isConnected,
    isConnecting: webSocket.isConnecting,
    
    // Actions
    ...actions,
    
    // WebSocket actions
    connect: webSocket.connect,
    disconnect: webSocket.disconnect,
    
    // Constants
    GAME_STATES,
    WS_STATES,
    MESSAGE_TYPES
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
};

// Custom hook to use WebSocket context
export const useWebSocketContext = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider');
  }
  return context;
};

export default WebSocketContext;