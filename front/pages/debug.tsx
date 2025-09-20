/**
 * Debug Page
 * Helps diagnose common issues with the application
 */

import React, { useState, useEffect } from 'react';
import { useAuthContext } from '../contexts/AuthContext';
import { useBalance } from '../hooks/useApi';
import { apiClient } from '../lib/api-client';
import { formatMNT } from '../lib/api-utils';

export default function DebugPage() {
  const { user, isAuthenticated, isLoading: authLoading } = useAuthContext();
  const { balance, error: balanceError, isLoading: balanceLoading } = useBalance();
  const [healthStatus, setHealthStatus] = useState<string>('Checking...');
  const [apiTest, setApiTest] = useState<string>('Not tested');
  const [tables, setTables] = useState<any[]>([]);
  const [tablesError, setTablesError] = useState<string | null>(null);

  useEffect(() => {
    // Test health endpoint
    apiClient.healthCheck()
      .then(() => setHealthStatus('✅ Healthy'))
      .catch((err) => setHealthStatus(`❌ Error: ${err.message}`));

    // Test tables endpoint
    apiClient.getTables()
      .then((response) => {
        setTables(response.tables);
        setApiTest('✅ API working');
      })
      .catch((err) => {
        setTablesError(err.message);
        setApiTest(`❌ API Error: ${err.message}`);
      });
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        <h1 className="text-3xl font-bold text-gray-900 mb-8">🔧 Debug Dashboard</h1>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Authentication Status */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">🔐 Authentication</h2>
            <div className="space-y-2">
              <p><strong>Loading:</strong> {authLoading ? '⏳ Yes' : '✅ No'}</p>
              <p><strong>Authenticated:</strong> {isAuthenticated ? '✅ Yes' : '❌ No'}</p>
              <p><strong>User ID:</strong> {user?.id || 'Not available'}</p>
              <p><strong>Username:</strong> {user?.username || 'Not available'}</p>
              <p><strong>Email:</strong> {user?.email || 'Not available'}</p>
              <p><strong>Role:</strong> {user?.role || 'Not available'}</p>
              <p><strong>Verified:</strong> {user?.is_verified ? '✅ Yes' : '❌ No'}</p>
            </div>
          </div>

          {/* Balance Status */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">💰 Balance</h2>
            <div className="space-y-2">
              <p><strong>Loading:</strong> {balanceLoading ? '⏳ Yes' : '✅ No'}</p>
              <p><strong>Error:</strong> {balanceError ? `❌ ${balanceError.message}` : '✅ None'}</p>
              {balance ? (
                <>
                  <p><strong>Main Balance:</strong> {formatMNT(balance.main_balance)}</p>
                  <p><strong>Game Balance:</strong> {formatMNT(balance.game_balance)}</p>
                  <p><strong>Total Balance:</strong> {formatMNT(balance.total_balance)}</p>
                </>
              ) : (
                <p><strong>Balance:</strong> ❌ Not available</p>
              )}
            </div>
          </div>

          {/* API Status */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">🌐 API Status</h2>
            <div className="space-y-2">
              <p><strong>Health Check:</strong> {healthStatus}</p>
              <p><strong>API Test:</strong> {apiTest}</p>
              <p><strong>Base URL:</strong> {process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}</p>
              <p><strong>Token Present:</strong> {typeof window !== 'undefined' && localStorage.getItem('auth_token') ? '✅ Yes' : '❌ No'}</p>
            </div>
          </div>

          {/* Tables Status */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">🎲 Tables</h2>
            <div className="space-y-2">
              <p><strong>Tables Loaded:</strong> {tables.length > 0 ? `✅ ${tables.length} tables` : '❌ No tables'}</p>
              <p><strong>Tables Error:</strong> {tablesError ? `❌ ${tablesError}` : '✅ None'}</p>
              {tables.length > 0 && (
                <div className="mt-4">
                  <h3 className="font-medium">Available Tables:</h3>
                  <ul className="text-sm mt-2 space-y-1">
                    {tables.slice(0, 3).map(table => (
                      <li key={table.id} className="text-gray-600">
                        • {table.name} ({table.current_players}/{table.max_players} players)
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Environment Info */}
        <div className="bg-white rounded-lg shadow-md p-6 mt-6">
          <h2 className="text-xl font-semibold mb-4">🔧 Environment</h2>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <p><strong>NODE_ENV:</strong> {process.env.NODE_ENV}</p>
            <p><strong>API_URL:</strong> {process.env.NEXT_PUBLIC_API_URL}</p>
            <p><strong>WS_URL:</strong> {process.env.NEXT_PUBLIC_WS_URL}</p>
            <p><strong>Current URL:</strong> {typeof window !== 'undefined' ? window.location.href : 'Server'}</p>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="bg-white rounded-lg shadow-md p-6 mt-6">
          <h2 className="text-xl font-semibold mb-4">⚡ Quick Actions</h2>
          <div className="flex flex-wrap gap-3">
            <button
              onClick={() => window.location.href = '/auth/login'}
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
            >
              Go to Login
            </button>
            <button
              onClick={() => window.location.href = '/dashboard'}
              className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700"
            >
              Go to Dashboard
            </button>
            <button
              onClick={() => window.location.href = '/lobby'}
              className="bg-purple-600 text-white px-4 py-2 rounded-md hover:bg-purple-700"
            >
              Go to Lobby
            </button>
            <button
              onClick={() => {
                if (typeof window !== 'undefined') {
                  localStorage.clear();
                  window.location.reload();
                }
              }}
              className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700"
            >
              Clear Storage & Reload
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}