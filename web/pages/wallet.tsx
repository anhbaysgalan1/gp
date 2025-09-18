/**
 * Wallet Page
 * Dedicated page for managing balance and transactions
 */

import React, { useState } from 'react';
import Link from 'next/link';
import { useAuthContext, withAuth } from '../contexts/AuthContext';
import { BalanceManager } from '../components/BalanceManager';
import { formatMNT } from '../lib/api-utils';
import { useBalance } from '../hooks/useApi';
import { apiClient } from '../lib/api-client';

function WalletPage() {
  const { user } = useAuthContext();
  const { balance, mutate: refreshBalance } = useBalance();
  const [addMoneyAmount, setAddMoneyAmount] = useState<string>('10000');
  const [isAddingMoney, setIsAddingMoney] = useState(false);
  const [addMoneyMessage, setAddMoneyMessage] = useState<string | null>(null);

  // Development function to add money using real backend deposit endpoint
  const handleAddMoney = async () => {
    const amount = parseInt(addMoneyAmount);
    if (!amount || amount <= 0) {
      setAddMoneyMessage('Please enter a valid amount');
      return;
    }

    if (!user?.id) {
      setAddMoneyMessage('User not authenticated');
      return;
    }

    setIsAddingMoney(true);
    setAddMoneyMessage(null);

    try {
      // Call the real backend deposit endpoint using API client
      const response = await apiClient.depositMoney(user.id, amount);

      if (response?.transaction_id) {
        // Refresh balance to show updated amount
        await refreshBalance();

        setAddMoneyMessage(`Successfully deposited ${formatMNT(amount)} to your account! Transaction ID: ${response.transaction_id}`);
        setAddMoneyAmount('10000'); // Reset to default
      } else {
        throw new Error('Invalid response from server');
      }
    } catch (error) {
      console.error('Error adding money:', error);
      const errorMessage = error instanceof Error ? error.message : 'Failed to add money. Please try again.';
      setAddMoneyMessage(errorMessage);
    } finally {
      setIsAddingMoney(false);
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
              <span className="text-gray-700">Wallet</span>
            </div>

            <div className="flex items-center space-x-4">
              {balance ? (
                <div className="bg-green-50 px-3 py-1 rounded-md">
                  <span className="text-sm text-green-700">
                    Total Balance: <span className="font-semibold">{formatMNT(balance.total_balance)}</span>
                  </span>
                </div>
              ) : (
                <div className="animate-pulse bg-gray-200 h-6 w-32 rounded"></div>
              )}

              <Link
                href="/dashboard"
                className="text-sm text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md hover:bg-gray-100"
              >
                Back to Dashboard
              </Link>

              <Link
                href="/lobby"
                className="text-sm text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md hover:bg-gray-100"
              >
                Game Lobby
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          {/* Header */}
          <div className="mb-6">
            <h1 className="text-3xl font-bold text-gray-900">Wallet</h1>
            <p className="mt-2 text-gray-600">
              Manage your balance, transfer funds, and view transaction history.
            </p>
          </div>

          {/* Balance Display */}
          <div className="bg-white rounded-lg shadow-md p-6 mb-6">
            <h2 className="text-xl font-semibold mb-4">Account Balance</h2>

            {balance ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="bg-blue-50 rounded-lg p-4">
                  <h3 className="text-sm font-medium text-gray-600 mb-1">Available Balance</h3>
                  <p className="text-3xl font-bold text-blue-600">
                    {formatMNT(balance.main_balance)}
                  </p>
                  <p className="text-sm text-gray-500 mt-1">Ready for games</p>
                </div>

                <div className="bg-green-50 rounded-lg p-4">
                  <h3 className="text-sm font-medium text-gray-600 mb-1">In Game</h3>
                  <p className="text-3xl font-bold text-green-600">
                    {formatMNT(balance.game_balance)}
                  </p>
                  <p className="text-sm text-gray-500 mt-1">Currently playing</p>
                </div>
              </div>
            ) : (
              <div className="animate-pulse">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="bg-gray-200 rounded-lg h-20"></div>
                  <div className="bg-gray-200 rounded-lg h-20"></div>
                </div>
              </div>
            )}
          </div>

          {/* Deposit/Withdraw Actions */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            {/* Deposit Section */}
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold mb-4 text-green-700">💰 Deposit Money</h3>

              <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3 mb-4">
                <p className="text-sm text-yellow-700">
                  <strong>Development Mode:</strong> For testing, you can add unlimited funds. In production, this will integrate with real payment providers.
                </p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Amount (MNT)
                  </label>
                  <input
                    type="number"
                    value={addMoneyAmount}
                    onChange={(e) => setAddMoneyAmount(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500"
                    placeholder="10000"
                    min="1000"
                    step="1000"
                    disabled={isAddingMoney}
                  />
                </div>

                <button
                  onClick={handleAddMoney}
                  disabled={isAddingMoney || !addMoneyAmount}
                  className="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isAddingMoney ? 'Processing...' : 'Deposit Money'}
                </button>

                {addMoneyMessage && (
                  <div className={`p-3 rounded-md text-sm ${
                    addMoneyMessage.includes('Successfully')
                      ? 'bg-green-100 text-green-700 border border-green-200'
                      : 'bg-red-100 text-red-700 border border-red-200'
                  }`}>
                    {addMoneyMessage}
                  </div>
                )}
              </div>
            </div>

            {/* Withdraw Section */}
            <div className="bg-white rounded-lg shadow-md p-6">
              <h3 className="text-lg font-semibold mb-4 text-blue-700">🏦 Withdraw Money</h3>

              <div className="bg-blue-50 border border-blue-200 rounded-md p-3 mb-4">
                <p className="text-sm text-blue-700">
                  <strong>Coming Soon:</strong> Withdrawal functionality will be available once payment gateway integration is complete.
                </p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Amount (MNT)
                  </label>
                  <input
                    type="number"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-100"
                    placeholder="Feature coming soon"
                    disabled
                  />
                </div>

                <button
                  disabled
                  className="w-full bg-gray-400 text-white py-2 px-4 rounded-md cursor-not-allowed"
                >
                  Withdraw Money (Coming Soon)
                </button>
              </div>
            </div>
          </div>

          {/* How It Works */}
          <div className="bg-white rounded-lg shadow-md p-6 mb-6">
            <h3 className="text-lg font-semibold mb-4">💡 How Your Wallet Works</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <h4 className="font-medium text-gray-900 mb-2">🎮 Playing Games</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• When you join a table, funds automatically move to "In Game"</li>
                  <li>• When you leave a table, winnings return to "Available Balance"</li>
                  <li>• All transactions are handled automatically</li>
                </ul>
              </div>

              <div>
                <h4 className="font-medium text-gray-900 mb-2">🔒 Security</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• All transactions are recorded in Formance ledger</li>
                  <li>• Double-entry bookkeeping ensures accuracy</li>
                  <li>• Real-time balance updates</li>
                </ul>
              </div>
            </div>
          </div>

          {/* Transaction History */}
          <div className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold mb-4">📊 Recent Transactions</h3>
            <BalanceManager />
          </div>
        </div>
      </div>
    </div>
  );
}

export default withAuth(WalletPage);