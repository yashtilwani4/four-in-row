import { useState, useEffect, useRef, useCallback } from 'react';

// WebSocket connection states
export const WS_STATES = {
  CONNECTING: 'connecting',
  CONNECTED: 'connected',
  DISCONNECTED: 'disconnected',
  RECONNECTING: 'reconnecting',
  ERROR: 'error',
  CLOSED: 'closed'
};

// Message types (matching backend)
export const MESSAGE_TYPES = {
  // Client to Server
  JOIN_QUEUE: 'join_queue',
  LEAVE_QUEUE: 'leave_queue',
  MAKE_MOVE: 'make_move',
  RECONNECT: 'reconnect',
  HEARTBEAT: 'heartbeat',
  GET_GAME_STATE: 'get_game_state',

  // Server to Client
  GAME_FOUND: 'game_found',
  GAME_STATE: 'game_state',
  MOVE_RESULT: 'move_result',
  GAME_END: 'game_end',
  ERROR: 'error',
  PLAYER_JOINED: 'player_joined',
  PLAYER_LEFT: 'player_left',
  TURN_CHANGED: 'turn_changed',
  HEARTBEAT_ACK: 'heartbeat_ack',
  BOT_MOVE: 'bot_move',
  RECONNECT_SUCCESS: 'reconnect_success',
  PLAYER_DISCONNECTED: 'player_disconnected',
  PLAYER_RECONNECTED: 'player_reconnected'
};

// Default configuration
const DEFAULT_CONFIG = {
  url: process.env.REACT_APP_WS_URL || (process.env.NODE_ENV === 'production' 
    ? `wss://${window.location.host}/ws`
    : 'ws://localhost:8080/ws'),
  reconnectInterval: 3000,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  connectionTimeout: 10000,
  enableLogging: process.env.NODE_ENV === 'development'
};

/**
 * Custom hook for WebSocket connection management
 * Provides automatic reconnection, message handling, and connection state management
 */
export const useWebSocket = (config = {}) => {
  const finalConfig = { ...DEFAULT_CONFIG, ...config };
  
  // Connection state
  const [connectionState, setConnectionState] = useState(WS_STATES.DISCONNECTED);
  const [lastError, setLastError] = useState(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const [isReconnecting, setIsReconnecting] = useState(false);
  
  // Connection statistics
  const [stats, setStats] = useState({
    messagesSent: 0,
    messagesReceived: 0,
    connectionTime: null,
    lastMessageTime: null,
    totalReconnects: 0
  });

  // Refs for managing connection and timers
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const heartbeatIntervalRef = useRef(null);
  const connectionTimeoutRef = useRef(null);
  const messageHandlersRef = useRef(new Map());
  const messageQueueRef = useRef([]);
  const isManualCloseRef = useRef(false);

  // Logging utility
  const log = useCallback((level, message, data = null) => {
    if (finalConfig.enableLogging) {
      console[level](`[WebSocket] ${message}`, data || '');
    }
  }, [finalConfig.enableLogging]);

  // Generate unique message ID
  const generateMessageId = useCallback(() => {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }, []);

  // Create WebSocket message
  const createMessage = useCallback((type, payload = null) => {
    return {
      type,
      payload,
      timestamp: new Date().toISOString(),
      message_id: generateMessageId()
    };
  }, [generateMessageId]);

  // Update connection statistics
  const updateStats = useCallback((update) => {
    setStats(prev => ({
      ...prev,
      ...update,
      lastMessageTime: new Date()
    }));
  }, []);

  // Clear all timers
  const clearTimers = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
    if (connectionTimeoutRef.current) {
      clearTimeout(connectionTimeoutRef.current);
      connectionTimeoutRef.current = null;
    }
  }, []);

  // Start heartbeat
  const startHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
    }

    heartbeatIntervalRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        const heartbeatMsg = createMessage(MESSAGE_TYPES.HEARTBEAT);
        wsRef.current.send(JSON.stringify(heartbeatMsg));
        log('debug', 'Heartbeat sent');
      }
    }, finalConfig.heartbeatInterval);
  }, [finalConfig.heartbeatInterval, createMessage, log]);

  // Handle WebSocket message
  const handleMessage = useCallback((event) => {
    try {
      const message = JSON.parse(event.data);
      log('debug', 'Message received', message);

      updateStats({ messagesReceived: stats.messagesReceived + 1 });

      // Handle system messages
      switch (message.type) {
        case MESSAGE_TYPES.HEARTBEAT_ACK:
          log('debug', 'Heartbeat acknowledged');
          return;
        
        case MESSAGE_TYPES.ERROR:
          setLastError(message.payload?.message || 'Unknown error');
          log('error', 'Server error', message.payload);
          break;
        
        case MESSAGE_TYPES.RECONNECT_SUCCESS:
          log('info', 'Reconnection successful', message.payload);
          setIsReconnecting(false);
          setReconnectAttempts(0);
          break;
      }

      // Call registered message handlers
      const handlers = messageHandlersRef.current.get(message.type) || [];
      handlers.forEach(handler => {
        try {
          handler(message);
        } catch (error) {
          log('error', 'Message handler error', error);
        }
      });

      // Call global message handler if registered
      const globalHandlers = messageHandlersRef.current.get('*') || [];
      globalHandlers.forEach(handler => {
        try {
          handler(message);
        } catch (error) {
          log('error', 'Global message handler error', error);
        }
      });

    } catch (error) {
      log('error', 'Failed to parse message', error);
      setLastError('Failed to parse server message');
    }
  }, [log, updateStats, stats.messagesReceived]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      log('warn', 'Already connected');
      return;
    }

    log('info', 'Connecting to WebSocket', finalConfig.url);
    setConnectionState(WS_STATES.CONNECTING);
    setLastError(null);
    isManualCloseRef.current = false;

    try {
      const ws = new WebSocket(finalConfig.url);
      wsRef.current = ws;

      // Connection timeout
      connectionTimeoutRef.current = setTimeout(() => {
        if (ws.readyState === WebSocket.CONNECTING) {
          log('error', 'Connection timeout');
          ws.close();
          setLastError('Connection timeout');
          setConnectionState(WS_STATES.ERROR);
        }
      }, finalConfig.connectionTimeout);

      ws.onopen = () => {
        log('info', 'WebSocket connected');
        clearTimeout(connectionTimeoutRef.current);
        setConnectionState(WS_STATES.CONNECTED);
        setLastError(null);
        setReconnectAttempts(0);
        setIsReconnecting(false);
        
        updateStats({ 
          connectionTime: new Date(),
          totalReconnects: isReconnecting ? stats.totalReconnects + 1 : stats.totalReconnects
        });

        // Start heartbeat
        startHeartbeat();

        // Send queued messages
        while (messageQueueRef.current.length > 0) {
          const queuedMessage = messageQueueRef.current.shift();
          ws.send(JSON.stringify(queuedMessage));
          log('debug', 'Sent queued message', queuedMessage);
        }
      };

      ws.onmessage = handleMessage;

      ws.onclose = (event) => {
        log('info', 'WebSocket closed', { code: event.code, reason: event.reason });
        clearTimers();
        
        if (!isManualCloseRef.current) {
          setConnectionState(WS_STATES.DISCONNECTED);
          
          // Attempt reconnection if not at max attempts
          if (reconnectAttempts < finalConfig.maxReconnectAttempts) {
            setIsReconnecting(true);
            setConnectionState(WS_STATES.RECONNECTING);
            
            reconnectTimeoutRef.current = setTimeout(() => {
              setReconnectAttempts(prev => prev + 1);
              connect();
            }, finalConfig.reconnectInterval);
            
            log('info', `Reconnecting in ${finalConfig.reconnectInterval}ms (attempt ${reconnectAttempts + 1}/${finalConfig.maxReconnectAttempts})`);
          } else {
            log('error', 'Max reconnection attempts reached');
            setConnectionState(WS_STATES.ERROR);
            setLastError('Max reconnection attempts reached');
          }
        } else {
          setConnectionState(WS_STATES.CLOSED);
        }
      };

      ws.onerror = (error) => {
        log('error', 'WebSocket error', error);
        setLastError('WebSocket connection error');
        setConnectionState(WS_STATES.ERROR);
      };

    } catch (error) {
      log('error', 'Failed to create WebSocket', error);
      setLastError('Failed to create WebSocket connection');
      setConnectionState(WS_STATES.ERROR);
    }
  }, [finalConfig.url, finalConfig.connectionTimeout, finalConfig.reconnectInterval, finalConfig.maxReconnectAttempts]); // Removed circular dependencies

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    log('info', 'Manually disconnecting');
    isManualCloseRef.current = true;
    clearTimers();
    
    if (wsRef.current) {
      wsRef.current.close(1000, 'Manual disconnect');
      wsRef.current = null;
    }
    
    setConnectionState(WS_STATES.CLOSED);
    setReconnectAttempts(0);
    setIsReconnecting(false);
  }, [log, clearTimers]);

  // Send message
  const sendMessage = useCallback((type, payload = null) => {
    const message = createMessage(type, payload);
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
      updateStats({ messagesSent: stats.messagesSent + 1 });
      log('debug', 'Message sent', message);
      return true;
    } else {
      // Queue message for later sending
      messageQueueRef.current.push(message);
      log('warn', 'Message queued (not connected)', message);
      return false;
    }
  }, [createMessage, updateStats, stats.messagesSent, log]);

  // Register message handler
  const onMessage = useCallback((messageType, handler) => {
    if (!messageHandlersRef.current.has(messageType)) {
      messageHandlersRef.current.set(messageType, []);
    }
    messageHandlersRef.current.get(messageType).push(handler);

    // Return unsubscribe function
    return () => {
      const handlers = messageHandlersRef.current.get(messageType) || [];
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    };
  }, []);

  // Game-specific helper methods
  const gameActions = {
    joinQueue: useCallback((playerName) => {
      return sendMessage(MESSAGE_TYPES.JOIN_QUEUE, { player_name: playerName });
    }, [sendMessage]),

    leaveQueue: useCallback(() => {
      return sendMessage(MESSAGE_TYPES.LEAVE_QUEUE);
    }, [sendMessage]),

    makeMove: useCallback((gameId, column) => {
      return sendMessage(MESSAGE_TYPES.MAKE_MOVE, { 
        game_id: gameId, 
        column: column 
      });
    }, [sendMessage]),

    reconnectToGame: useCallback((gameId, playerId, username) => {
      return sendMessage(MESSAGE_TYPES.RECONNECT, {
        game_id: gameId,
        player_id: playerId,
        username: username,
        last_seen: new Date().toISOString()
      });
    }, [sendMessage]),

    getGameState: useCallback((gameId) => {
      return sendMessage(MESSAGE_TYPES.GET_GAME_STATE, { game_id: gameId });
    }, [sendMessage])
  };

  // Initialize connection on mount
  useEffect(() => {
    connect();
    
    return () => {
      disconnect();
    };
  }, []); // Remove dependencies to prevent reconnection loops

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      clearTimers();
      messageHandlersRef.current.clear();
      messageQueueRef.current = [];
    };
  }, [clearTimers]);

  return {
    // Connection state
    connectionState,
    isConnected: connectionState === WS_STATES.CONNECTED,
    isConnecting: connectionState === WS_STATES.CONNECTING,
    isReconnecting,
    lastError,
    reconnectAttempts,
    
    // Statistics
    stats,
    
    // Connection control
    connect,
    disconnect,
    
    // Message handling
    sendMessage,
    onMessage,
    
    // Game actions
    ...gameActions,
    
    // Constants
    MESSAGE_TYPES,
    WS_STATES
  };
};