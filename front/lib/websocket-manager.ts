/**
 * Singleton WebSocket Manager
 * Manages a single WebSocket connection outside of React lifecycle
 * Provides centralized connection state and message handling
 */

import { config } from './config';

export interface WebSocketMessage {
  action: string;
  [key: string]: any;
}

export interface WebSocketEventHandlers {
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
  onMessage?: (message: WebSocketMessage) => void;
}

class WebSocketManager {
  private socket: WebSocket | null = null;
  private isConnecting = false;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private eventHandlers: Set<WebSocketEventHandlers> = new Set();
  private messageQueue: WebSocketMessage[] = [];
  private authToken: string | null = null;

  constructor() {
    // Singleton pattern
    if (typeof window !== 'undefined') {
      // Store instance on window to ensure singleton across hot reloads
      if ((window as any).__wsManager) {
        return (window as any).__wsManager;
      }
      (window as any).__wsManager = this;
    }
  }

  /**
   * Initialize WebSocket connection with authentication
   */
  public async connect(token?: string): Promise<void> {
    if (typeof window === 'undefined') {
      console.warn('WebSocket can only be initialized in browser environment');
      return;
    }

    // Update auth token if provided
    if (token) {
      this.authToken = token;
    } else {
      // Try to get token from localStorage
      this.authToken = localStorage.getItem('auth_token');
    }

    if (!this.authToken) {
      console.warn('No auth token found, WebSocket connection will fail');
      return;
    }

    // Prevent multiple simultaneous connections
    if (this.isConnecting || this.isConnected()) {
      console.log('WebSocket already connecting or connected, skipping');
      return;
    }

    this.isConnecting = true;

    try {
      const wsUrl = `${config.websocket.url}/ws?token=${encodeURIComponent(this.authToken)}`;
      console.log('Creating WebSocket connection to:', wsUrl);

      this.socket = new WebSocket(wsUrl);
      this.setupEventHandlers();

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.isConnecting = false;
      throw error;
    }
  }

  /**
   * Setup WebSocket event handlers
   */
  private setupEventHandlers(): void {
    if (!this.socket) return;

    this.socket.onopen = () => {
      console.log('WebSocket connected successfully');
      this.isConnecting = false;
      this.reconnectAttempts = 0;

      // Process queued messages
      this.processMessageQueue();

      // Notify all registered handlers
      this.eventHandlers.forEach(handler => {
        if (handler.onOpen) {
          handler.onOpen();
        }
      });
    };

    this.socket.onclose = (event) => {
      console.log('WebSocket disconnected:', event.code, event.reason);
      this.isConnecting = false;

      // Notify all registered handlers
      this.eventHandlers.forEach(handler => {
        if (handler.onClose) {
          handler.onClose();
        }
      });

      // Attempt reconnection if not a clean close
      if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.scheduleReconnect();
      }
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.isConnecting = false;

      // Notify all registered handlers
      this.eventHandlers.forEach(handler => {
        if (handler.onError) {
          handler.onError(error);
        }
      });
    };

    this.socket.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);

        // Notify all registered handlers
        this.eventHandlers.forEach(handler => {
          if (handler.onMessage) {
            handler.onMessage(message);
          }
        });

      } catch (error) {
        console.error('Failed to parse WebSocket message:', error, event.data);
      }
    };
  }

  /**
   * Schedule automatic reconnection
   */
  private scheduleReconnect(): void {
    this.reconnectAttempts++;
    const delay = this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1);

    console.log(`Scheduling WebSocket reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);

    setTimeout(() => {
      if (!this.isConnected() && this.authToken) {
        console.log('Attempting WebSocket reconnection...');
        this.connect();
      }
    }, delay);
  }

  /**
   * Process queued messages when connection becomes available
   */
  private processMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.isConnected()) {
      const message = this.messageQueue.shift();
      if (message) {
        this.sendMessage(message);
      }
    }
  }

  /**
   * Send message through WebSocket
   */
  public sendMessage(message: WebSocketMessage): boolean {
    if (!this.isConnected()) {
      console.warn('WebSocket not connected, queueing message:', message);
      this.messageQueue.push(message);
      return false;
    }

    try {
      this.socket!.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('Failed to send WebSocket message:', error);
      return false;
    }
  }

  /**
   * Register event handlers
   */
  public addEventHandler(handlers: WebSocketEventHandlers): () => void {
    this.eventHandlers.add(handlers);

    // Return unsubscribe function
    return () => {
      this.eventHandlers.delete(handlers);
    };
  }

  /**
   * Check if WebSocket is connected
   */
  public isConnected(): boolean {
    return this.socket !== null && this.socket.readyState === WebSocket.OPEN;
  }

  /**
   * Check if WebSocket is connecting
   */
  public isConnectingState(): boolean {
    return this.isConnecting || (this.socket !== null && this.socket.readyState === WebSocket.CONNECTING);
  }

  /**
   * Get current connection state
   */
  public getConnectionState(): string {
    if (!this.socket) return 'DISCONNECTED';

    switch (this.socket.readyState) {
      case WebSocket.CONNECTING:
        return 'CONNECTING';
      case WebSocket.OPEN:
        return 'CONNECTED';
      case WebSocket.CLOSING:
        return 'CLOSING';
      case WebSocket.CLOSED:
        return 'CLOSED';
      default:
        return 'UNKNOWN';
    }
  }

  /**
   * Disconnect WebSocket
   */
  public disconnect(): void {
    if (this.socket) {
      console.log('Disconnecting WebSocket');
      this.socket.close(1000, 'Client disconnecting');
      this.socket = null;
    }
    this.isConnecting = false;
    this.reconnectAttempts = 0;
    this.messageQueue = [];
  }

  /**
   * Force reconnection
   */
  public reconnect(): void {
    this.disconnect();
    if (this.authToken) {
      this.connect();
    }
  }

  /**
   * Update authentication token
   */
  public updateAuthToken(token: string): void {
    this.authToken = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token);
    }
  }

  /**
   * Get message queue length for debugging
   */
  public getQueueLength(): number {
    return this.messageQueue.length;
  }

  /**
   * Get detailed connection statistics for monitoring
   */
  public getConnectionStats() {
    return {
      isConnected: this.isConnected(),
      isConnecting: this.isConnectingState(),
      connectionState: this.getConnectionState(),
      reconnectAttempts: this.reconnectAttempts,
      maxReconnectAttempts: this.maxReconnectAttempts,
      queueLength: this.messageQueue.length,
      hasAuthToken: !!this.authToken,
      eventHandlersCount: this.eventHandlers.size
    };
  }

  /**
   * Add connection health monitoring
   */
  public isHealthy(): boolean {
    // Connection is healthy if it's connected or connecting with reasonable attempts
    return this.isConnected() ||
           (this.isConnectingState() && this.reconnectAttempts < this.maxReconnectAttempts);
  }

  /**
   * Clear message queue (useful for debugging)
   */
  public clearQueue(): number {
    const cleared = this.messageQueue.length;
    this.messageQueue = [];
    return cleared;
  }

  /**
   * Enhanced error handling with detailed logging
   */
  private handleConnectionError(error: Event, context: string) {
    console.error(`WebSocket ${context} error:`, {
      error,
      connectionState: this.getConnectionState(),
      reconnectAttempts: this.reconnectAttempts,
      queueLength: this.messageQueue.length,
      hasAuthToken: !!this.authToken,
      timestamp: new Date().toISOString()
    });

    // Notify error handlers
    this.eventHandlers.forEach(handler => {
      if (handler.onError) {
        handler.onError(error);
      }
    });
  }
}

// Export singleton instance
export const wsManager = new WebSocketManager();