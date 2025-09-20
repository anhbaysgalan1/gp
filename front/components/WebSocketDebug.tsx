/**
 * WebSocket Debug Component
 * Shows real-time WebSocket connection state for development/testing
 */

import React from 'react';
import { useWebSocket } from '../contexts/WebSocketContext';

interface WebSocketDebugProps {
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
  expanded?: boolean;
}

export function WebSocketDebug({ position = 'bottom-right', expanded = false }: WebSocketDebugProps) {
  const { isConnected, isConnecting, connectionState, queueLength } = useWebSocket();

  const [isExpanded, setIsExpanded] = React.useState(expanded);

  const positionClasses = {
    'top-left': 'top-4 left-4',
    'top-right': 'top-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'bottom-right': 'bottom-4 right-4'
  };

  const getStatusColor = () => {
    if (isConnected) return 'bg-green-500';
    if (isConnecting) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  const getStatusText = () => {
    if (isConnected) return 'Connected';
    if (isConnecting) return 'Connecting';
    return 'Disconnected';
  };

  if (!isExpanded) {
    return (
      <div className={`fixed ${positionClasses[position]} z-50`}>
        <button
          onClick={() => setIsExpanded(true)}
          className={`${getStatusColor()} rounded-full w-4 h-4 border-2 border-white shadow-lg cursor-pointer hover:scale-110 transition-transform`}
          title={`WebSocket: ${getStatusText()}`}
        />
      </div>
    );
  }

  return (
    <div className={`fixed ${positionClasses[position]} z-50`}>
      <div className="bg-black bg-opacity-80 text-white p-3 rounded-lg text-xs font-mono shadow-lg">
        <div className="flex items-center justify-between mb-2">
          <span className="font-bold">WebSocket Status</span>
          <button
            onClick={() => setIsExpanded(false)}
            className="text-gray-400 hover:text-white ml-2"
          >
            ×
          </button>
        </div>

        <div className="space-y-1">
          <div className="flex items-center space-x-2">
            <div className={`w-2 h-2 rounded-full ${getStatusColor()}`} />
            <span>{getStatusText()}</span>
          </div>

          <div>State: {connectionState}</div>

          {queueLength > 0 && (
            <div className="text-yellow-400">
              Queue: {queueLength} message{queueLength !== 1 ? 's' : ''}
            </div>
          )}

          <div className="text-gray-400 text-xs mt-2">
            {new Date().toLocaleTimeString()}
          </div>
        </div>
      </div>
    </div>
  );
}

export default WebSocketDebug;