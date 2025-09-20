/**
 * Unified WebSocket Context
 * Provides centralized WebSocket state management using the singleton manager
 * Handles all game state updates and integrates with AppStore
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { useAuthContext } from './AuthContext';
import { useSessionContext } from './SessionContext';
import { AppContext } from '../providers/AppStore';
import { wsManager, WebSocketMessage, WebSocketEventHandlers } from '../lib/websocket-manager';
import { Message, Game, Log } from '../interfaces';
import { SessionInfo } from '../types/api';

interface WebSocketContextState {
  isConnected: boolean;
  isConnecting: boolean;
  connectionState: string;
  sendMessage: (message: WebSocketMessage) => boolean;
  reconnect: () => void;
  queueLength: number;
  connectionStats: () => any;
  isHealthy: () => boolean;
}

const WebSocketContext = createContext<WebSocketContextState | null>(null);

interface WebSocketProviderProps {
  children: ReactNode;
}

export function WebSocketProvider({ children }: WebSocketProviderProps) {
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectionState, setConnectionState] = useState('DISCONNECTED');
  const [queueLength, setQueueLength] = useState(0);

  const { user } = useAuthContext();
  const { setSessionInfo } = useSessionContext();
  const { appState, dispatch } = useContext(AppContext);

  // Setup WebSocket event handlers for game state management
  useEffect(() => {
    const handlers: WebSocketEventHandlers = {
      onOpen: () => {
        console.log('WebSocket Context: Connection opened');
        setIsConnected(true);
        setIsConnecting(false);
        setConnectionState('CONNECTED');
        setQueueLength(0);

        // Set username from auth context
        if (user && user.username) {
          dispatch({ type: "setUsername", payload: user.username });
        }
      },

      onClose: () => {
        console.log('WebSocket Context: Connection closed');
        setIsConnected(false);
        setIsConnecting(false);
        setConnectionState('CLOSED');
      },

      onError: (error) => {
        console.error('WebSocket Context: Connection error', error);
        setIsConnected(false);
        setIsConnecting(false);
        setConnectionState('ERROR');
      },

      onMessage: (message: WebSocketMessage) => {
        console.log('WebSocket Context: Received message', message.action);
        handleWebSocketMessage(message);
        setQueueLength(wsManager.getQueueLength());
      }
    };

    // Register handlers with the singleton manager
    const unsubscribe = wsManager.addEventHandler(handlers);

    // Update connection state periodically
    const stateUpdateInterval = setInterval(() => {
      setIsConnected(wsManager.isConnected());
      setIsConnecting(wsManager.isConnectingState());
      setConnectionState(wsManager.getConnectionState());
      setQueueLength(wsManager.getQueueLength());
    }, 1000);

    return () => {
      unsubscribe();
      clearInterval(stateUpdateInterval);
    };
  }, [user, dispatch, setSessionInfo]);

  // Initialize WebSocket connection when user is available
  useEffect(() => {
    if (user && !wsManager.isConnected() && !wsManager.isConnectingState()) {
      const token = localStorage.getItem('auth_token');
      if (token) {
        console.log('WebSocket Context: Initializing connection for user', user.username);
        setIsConnecting(true);
        wsManager.connect(token).catch(error => {
          console.error('WebSocket Context: Failed to connect', error);
          setIsConnecting(false);
        });
      }
    }
  }, [user]);

  // Handle WebSocket messages and update game state
  const handleWebSocketMessage = (event: WebSocketMessage) => {
    switch (event.action) {
      case "new-message":
        const newMessage: Message = {
          name: event.username,
          message: event.message,
          timestamp: event.timestamp,
        };
        dispatch({ type: "addMessage", payload: newMessage });
        break;

      case "new-log":
        const newLog: Log = {
          message: event.message,
          timestamp: event.timestamp,
        };
        dispatch({ type: "addLog", payload: newLog });
        break;

      case "update-game":
        const newGame: Game = {
          running: event.game.running,
          dealer: event.game.dealer,
          action: event.game.action,
          utg: event.game.utg,
          sb: event.game.sb,
          bb: event.game.bb,
          communityCards: event.game.communityCards,
          stage: event.game.stage,
          betting: event.game.betting,
          config: event.game.config,
          players: event.game.players,
          pots: event.game.pots,
          minRaise: event.game.minRaise,
          readyCount: event.game.readyCount,
        };
        dispatch({ type: "updateGame", payload: newGame });

        // Handle session info update from WebSocket
        if (event.session_info) {
          setSessionInfo(event.session_info);
          console.log("Session info updated:", event.session_info);
        } else {
          // Clear session info if not provided
          setSessionInfo(null);
        }
        break;

      case "update-player-uuid":
        dispatch({ type: "updatePlayerID", payload: event.uuid });
        break;

      case "error":
        console.error("WebSocket error:", event.message);
        // TODO: Replace with proper toast notification
        alert(event.message);
        break;

      case "success":
        console.log("WebSocket success:", event.message);
        // TODO: Replace with proper toast notification
        break;

      case "update-balance":
        const balanceUpdate = {
          main_balance: event.main_balance,
          game_balance: event.game_balance,
          total_balance: event.main_balance + event.game_balance,
        };
        console.log("Balance update received:", balanceUpdate);

        // Trigger SWR cache update for balance
        if (typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('balance-update', {
            detail: balanceUpdate
          }));
        }
        break;

      default:
        console.warn("Unknown WebSocket message:", event);
        break;
    }
  };

  // Send message through WebSocket manager
  const sendMessage = (message: WebSocketMessage): boolean => {
    const sent = wsManager.sendMessage(message);
    setQueueLength(wsManager.getQueueLength());
    return sent;
  };

  // Force reconnection
  const reconnect = () => {
    console.log('WebSocket Context: Manual reconnection requested');
    setIsConnecting(true);
    wsManager.reconnect();
  };

  const contextValue: WebSocketContextState = {
    isConnected,
    isConnecting,
    connectionState,
    sendMessage,
    reconnect,
    queueLength,
    connectionStats: () => wsManager.getConnectionStats(),
    isHealthy: () => wsManager.isHealthy()
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
}

// Custom hook to use WebSocket context
export function useWebSocket(): WebSocketContextState {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}

// Custom hook for table-specific WebSocket operations
export function useTableWebSocket(tableId?: string) {
  const { sendMessage, isConnected } = useWebSocket();

  // Join table when tableId is provided and connection is available
  useEffect(() => {
    if (isConnected && tableId) {
      console.log("Joining table via WebSocket:", tableId);

      // Small delay to ensure WebSocket is fully ready
      const timeoutId = setTimeout(() => {
        sendMessage({
          action: "join-table",
          tablename: tableId
        });

        // Request balance update
        sendMessage({
          action: "get-balance"
        });
      }, 100);

      return () => clearTimeout(timeoutId);
    }
  }, [isConnected, tableId, sendMessage]);

  return {
    sendMessage,
    isConnected,
    joinTable: (tableName: string) => sendMessage({
      action: "join-table",
      tablename: tableName
    }),
    leaveTable: (tableName: string) => sendMessage({
      action: "leave-table",
      tablename: tableName
    })
  };
}