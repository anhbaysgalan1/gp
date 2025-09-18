/**
 * Dynamic Game Page
 * Renders the poker game interface for a specific table
 */

import { useRouter } from 'next/router';
import React, { useEffect, useState } from 'react';
import { useAuthContext, withAuth } from '../../contexts/AuthContext';
import { useSessionContext } from '../../contexts/SessionContext';
import { apiClient } from '../../lib/api-client';
import { PokerTable } from '../../types/api';
import Game from '../../components/Game';

function GamePage() {
  const router = useRouter();
  const { user } = useAuthContext();
  const { isSeated, hasSession, seatNumber } = useSessionContext();
  const { tableId } = router.query;

  const [table, setTable] = useState<PokerTable | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch table details
  useEffect(() => {
    const fetchTable = async () => {
      if (!tableId || typeof tableId !== 'string') return;

      try {
        setLoading(true);
        setError(null);

        const tableData = await apiClient.getTable(tableId);
        setTable(tableData);

      } catch (err) {
        console.error('Failed to fetch table:', err);
        setError('Failed to load table. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    fetchTable();
  }, [tableId]);


  const handleLeaveTable = async () => {
    if (!tableId || typeof tableId !== 'string') return;

    try {
      await apiClient.leaveTable(tableId);
      router.push('/lobby');
    } catch (err) {
      console.error('Failed to leave table:', err);
      setError('Failed to leave table. Please try again.');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-white mx-auto"></div>
          <p className="mt-4 text-white">Loading game...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white mb-4">Error</h1>
          <p className="text-red-400 mb-6">{error}</p>
          <button
            onClick={() => router.push('/lobby')}
            className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
          >
            Back to Lobby
          </button>
        </div>
      </div>
    );
  }

  if (!table) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white mb-4">Table Not Found</h1>
          <p className="text-gray-400 mb-6">The requested table could not be found.</p>
          <button
            onClick={() => router.push('/lobby')}
            className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
          >
            Back to Lobby
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="relative bg-gray-900 min-h-screen">
      {/* Game Header */}
      <div className="absolute top-0 left-0 right-0 z-10 bg-black bg-opacity-50 p-4">
        <div className="flex justify-between items-center">
          <div className="text-white">
            <h1 className="text-xl font-bold">{table.name}</h1>
            <p className="text-sm text-gray-300">
              {table.game_type.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())} •
              Blinds: {table.small_blind}/{table.big_blind} MNT •
              Players: {table.current_players}/{table.max_players}
            </p>
          </div>

          <div className="flex space-x-3">
            {hasSession && (
              <button
                onClick={handleLeaveTable}
                className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700 text-sm"
              >
                Leave Table
              </button>
            )}

            <button
              onClick={() => router.push('/lobby')}
              className="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700 text-sm"
            >
              Back to Lobby
            </button>
          </div>
        </div>
      </div>

      {/* Game Component */}
      <div className="pt-20">
        <Game />
      </div>

      {/* Session Info (for debugging) */}
      <div className="absolute bottom-4 left-4 bg-black bg-opacity-50 text-white p-2 rounded text-xs">
        <div>Has Session: {hasSession ? 'Yes' : 'No'}</div>
        <div>Is Seated: {isSeated ? 'Yes' : 'No'}</div>
        <div>Seat: {seatNumber || 'None'}</div>
      </div>

    </div>
  );
}

export default withAuth(GamePage);