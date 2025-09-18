/**
 * Lobby Page
 * Shows available poker tables and allows users to join games
 */

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useAuthContext, withAuth } from '../contexts/AuthContext';
import { formatMNT } from '../lib/api-utils';
import { apiClient } from '../lib/api-client';
import { PokerTable } from '../types/api';

function LobbyPage() {
  const { user } = useAuthContext();
  const [tables, setTables] = useState<PokerTable[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch tables from backend
  useEffect(() => {
    const fetchTables = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await apiClient.getTables({
          limit: 20
          // Remove status filter to see all tables initially
        });
        setTables(response.tables);
      } catch (err) {
        console.error('Failed to fetch tables:', err);
        setError('Failed to load tables. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    fetchTables();
  }, []);

  const handleEnterTable = (tableId: string) => {
    // Direct navigation to table view - no buy-in modal needed
    window.location.href = `/game/${tableId}`;
  };


  const getStakesLevel = (smallBlind: number) => {
    if (smallBlind <= 100) return 'Low';
    if (smallBlind <= 1000) return 'Medium';
    return 'High';
  };

  const getStakesColor = (stakes: string) => {
    switch (stakes) {
      case 'Low': return 'text-green-600 bg-green-100';
      case 'Medium': return 'text-yellow-600 bg-yellow-100';
      case 'High': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center space-x-4">
              <Link href="/dashboard" className="text-xl font-bold text-gray-900">
                🃏 Go Poker Platform
              </Link>
              <span className="text-gray-500">/</span>
              <span className="text-gray-700">Lobby</span>
            </div>

            <div className="flex items-center space-x-4">
              <Link
                href="/wallet"
                className="text-sm text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md hover:bg-gray-100"
              >
                Wallet
              </Link>
              <Link
                href="/dashboard"
                className="text-sm text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md hover:bg-gray-100"
              >
                Back to Dashboard
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          {/* Header */}
          <div className="mb-6 flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Game Lobby</h1>
              <p className="mt-2 text-gray-600">
                Choose a table and join the action. Make sure you have enough balance for the buy-in.
              </p>
            </div>
            <Link
              href="/create-table"
              className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 text-sm font-medium"
            >
              Create Table
            </Link>
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          {/* Loading State */}
          {loading ? (
            <div className="text-center py-12">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
              <p className="mt-4 text-gray-600">Loading tables...</p>
            </div>
          ) : (
            <>
              {/* Tables Grid */}
              <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
                {tables.map((table) => {
                  const stakesLevel = getStakesLevel(table.small_blind);
                  return (
              <div key={table.id} className="bg-white rounded-lg shadow-md overflow-hidden">
                <div className="p-6">
                  <div className="flex justify-between items-start mb-4">
                    <h3 className="text-lg font-semibold text-gray-900">{table.name}</h3>
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStakesColor(stakesLevel)}`}>
                      {stakesLevel} Stakes
                    </span>
                  </div>

                  <div className="space-y-3">
                    <div className="flex justify-between">
                      <span className="text-sm text-gray-600">Game Type:</span>
                      <span className="text-sm font-medium">
                        {table.game_type.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                      </span>
                    </div>

                    <div className="flex justify-between">
                      <span className="text-sm text-gray-600">Players:</span>
                      <span className="text-sm font-medium">
                        {table.current_players}/{table.max_players}
                      </span>
                    </div>

                    <div className="flex justify-between">
                      <span className="text-sm text-gray-600">Blinds:</span>
                      <span className="text-sm font-medium">
                        {formatMNT(table.small_blind)}/{formatMNT(table.big_blind)}
                      </span>
                    </div>

                    <div className="flex justify-between">
                      <span className="text-sm text-gray-600">Buy-in Range:</span>
                      <span className="text-sm font-medium">
                        {formatMNT(table.min_buy_in)} - {formatMNT(table.max_buy_in)}
                      </span>
                    </div>
                  </div>

                  <div className="mt-6">
                    <button
                      onClick={() => handleEnterTable(table.id)}
                      className="w-full py-2 px-4 rounded-md text-sm font-medium bg-blue-600 text-white hover:bg-blue-700"
                    >
                      Enter Table
                    </button>
                  </div>
                </div>

                {/* Players indicator */}
                <div className="bg-gray-50 px-6 py-3">
                  <div className="flex justify-between items-center">
                    <span className="text-xs text-gray-500">Available Seats</span>
                    <div className="flex space-x-1">
                      {Array.from({ length: table.max_players }).map((_, index) => (
                        <div
                          key={index}
                          className={`w-3 h-3 rounded-full ${
                            index < table.current_players ? 'bg-blue-600' : 'bg-gray-300'
                          }`}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </div>
                  );
                })}
          </div>

              {/* Empty State */}
              {tables.length === 0 && (
                <div className="text-center py-12">
                  <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM9 9a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                  <h3 className="mt-2 text-sm font-medium text-gray-900">No tables available</h3>
                  <p className="mt-1 text-sm text-gray-500">
                    All tables are currently full. Check back later or create a new table.
                  </p>
                </div>
              )}
            </>
          )}
        </div>
      </div>

    </div>
  );
}

export default withAuth(LobbyPage);