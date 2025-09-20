/**
 * Session Context for managing game session state
 * Provides session information received from WebSocket updates
 */

import React, { createContext, useContext, useState, ReactNode } from 'react';
import { SessionInfo } from '../types/api';

interface SessionContextType {
  sessionInfo: SessionInfo | null;
  setSessionInfo: (sessionInfo: SessionInfo | null) => void;
  isSeated: boolean;
  hasSession: boolean;
  seatNumber?: number;
}

const SessionContext = createContext<SessionContextType | undefined>(undefined);

interface SessionProviderProps {
  children: ReactNode;
}

export function SessionProvider({ children }: SessionProviderProps) {
  const [sessionInfo, setSessionInfo] = useState<SessionInfo | null>(null);

  const contextValue: SessionContextType = {
    sessionInfo,
    setSessionInfo,
    isSeated: sessionInfo?.is_seated || false,
    hasSession: sessionInfo?.has_session || false,
    seatNumber: sessionInfo?.seat_number,
  };

  return (
    <SessionContext.Provider value={contextValue}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSessionContext(): SessionContextType {
  const context = useContext(SessionContext);
  if (context === undefined) {
    throw new Error('useSessionContext must be used within a SessionProvider');
  }
  return context;
}