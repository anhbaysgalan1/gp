/**
 * Create Table Page
 * Allows authenticated users to create new poker tables
 */

import React, { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useAuthContext, withAuth } from '../contexts/AuthContext';
import { apiClient } from '../lib/api-client';
import { CreateTableRequest } from '../types/api';

function CreateTablePage() {
  const router = useRouter();
  const { user } = useAuthContext();

  const [formData, setFormData] = useState<CreateTableRequest>({
    name: '',
    table_type: 'cash',
    game_type: 'texas_holdem',
    max_players: 9,
    min_buy_in: 1000,
    max_buy_in: 10000,
    small_blind: 50,
    big_blind: 100,
    is_private: false,
    password: ''
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;

    if (type === 'checkbox') {
      const checked = (e.target as HTMLInputElement).checked;
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else if (type === 'number') {
      setFormData(prev => ({ ...prev, [name]: parseInt(value) || 0 }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      // Validation
      if (!formData.name.trim()) {
        throw new Error('Table name is required');
      }

      if (formData.max_buy_in <= formData.min_buy_in) {
        throw new Error('Max buy-in must be greater than min buy-in');
      }

      if (formData.big_blind <= formData.small_blind) {
        throw new Error('Big blind must be greater than small blind');
      }

      const table = await apiClient.createTable(formData);
      console.log('Table created:', table);

      // Redirect to lobby
      router.push('/lobby');
    } catch (err) {
      console.error('Failed to create table:', err);
      setError(err instanceof Error ? err.message : 'Failed to create table');
    } finally {
      setIsSubmitting(false);
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
              <span className="text-gray-700">Create Table</span>
            </div>

            <div className="flex items-center space-x-4">
              <Link
                href="/lobby"
                className="text-sm text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md hover:bg-gray-100"
              >
                Back to Lobby
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="max-w-2xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="bg-white rounded-lg shadow p-6">
            <h1 className="text-2xl font-bold text-gray-900 mb-6">Create New Table</h1>

            {error && (
              <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Table Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Table Name *
                </label>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter table name"
                  required
                />
              </div>

              {/* Table Type and Game Type */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Table Type
                  </label>
                  <select
                    name="table_type"
                    value={formData.table_type}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="cash">Cash Game</option>
                    <option value="tournament">Tournament</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Game Type
                  </label>
                  <select
                    name="game_type"
                    value={formData.game_type}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="texas_holdem">Texas Hold'em</option>
                    <option value="omaha">Omaha</option>
                    <option value="stud">Seven Card Stud</option>
                  </select>
                </div>
              </div>

              {/* Max Players */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Maximum Players
                </label>
                <select
                  name="max_players"
                  value={formData.max_players}
                  onChange={handleInputChange}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={2}>2 Players (Heads Up)</option>
                  <option value={6}>6 Players</option>
                  <option value={9}>9 Players</option>
                  <option value={10}>10 Players</option>
                </select>
              </div>

              {/* Buy-in Range */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Min Buy-in (MNT) *
                  </label>
                  <input
                    type="number"
                    name="min_buy_in"
                    value={formData.min_buy_in}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="100"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Max Buy-in (MNT) *
                  </label>
                  <input
                    type="number"
                    name="max_buy_in"
                    value={formData.max_buy_in}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="100"
                    required
                  />
                </div>
              </div>

              {/* Blinds */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Small Blind (MNT) *
                  </label>
                  <input
                    type="number"
                    name="small_blind"
                    value={formData.small_blind}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Big Blind (MNT) *
                  </label>
                  <input
                    type="number"
                    name="big_blind"
                    value={formData.big_blind}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1"
                    required
                  />
                </div>
              </div>

              {/* Private Table */}
              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    name="is_private"
                    checked={formData.is_private}
                    onChange={handleInputChange}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <span className="ml-2 text-sm text-gray-700">Private Table</span>
                </label>
              </div>

              {/* Password (if private) */}
              {formData.is_private && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Table Password
                  </label>
                  <input
                    type="password"
                    name="password"
                    value={formData.password}
                    onChange={handleInputChange}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Enter password for private table"
                  />
                </div>
              )}

              {/* Submit Button */}
              <div className="flex space-x-3 pt-4">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className={`flex-1 py-2 px-4 rounded-md font-medium ${
                    isSubmitting
                      ? 'bg-gray-400 text-gray-700 cursor-not-allowed'
                      : 'bg-blue-600 text-white hover:bg-blue-700'
                  }`}
                >
                  {isSubmitting ? 'Creating Table...' : 'Create Table'}
                </button>
                <Link
                  href="/lobby"
                  className="flex-1 bg-gray-300 text-gray-700 py-2 px-4 rounded-md hover:bg-gray-400 text-center"
                >
                  Cancel
                </Link>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}

export default withAuth(CreateTablePage);