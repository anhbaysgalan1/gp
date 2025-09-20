/**
 * Balance Manager Component
 * Enhanced transaction history with filtering and pagination
 */

import React, { useState, useMemo } from 'react';
import { useTransactionHistory } from '../hooks/useApi';
import { formatMNT, formatDateTime, getErrorMessage } from '../lib/api-utils';
import { Transaction, TransactionType } from '../types/api';

interface BalanceManagerProps {
  sessionId?: string;
}

interface TransactionFilters {
  type: TransactionType | 'all';
  dateRange: 'all' | 'today' | 'week' | 'month' | 'custom';
  amountRange: 'all' | 'small' | 'medium' | 'large' | 'custom';
  searchText: string;
  customDateFrom?: string;
  customDateTo?: string;
  customAmountMin?: number;
  customAmountMax?: number;
}

const initialFilters: TransactionFilters = {
  type: 'all',
  dateRange: 'all',
  amountRange: 'all',
  searchText: '',
};

export function BalanceManager({ sessionId = 'demo-session' }: BalanceManagerProps) {
  const [filters, setFilters] = useState<TransactionFilters>(initialFilters);
  const [showFilters, setShowFilters] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  const { transactions, error: transactionError, isLoading: transactionLoading } = useTransactionHistory(50, 0);

  // Filter transactions based on current filters
  const filteredTransactions = useMemo(() => {
    if (!transactions) return [];

    return transactions.filter((transaction) => {
      // Type filter
      if (filters.type !== 'all' && transaction.type !== filters.type) {
        return false;
      }

      // Date range filter
      if (filters.dateRange !== 'all') {
        const transactionDate = new Date(transaction.created_at);
        const now = new Date();

        switch (filters.dateRange) {
          case 'today':
            if (transactionDate.toDateString() !== now.toDateString()) return false;
            break;
          case 'week':
            const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
            if (transactionDate < weekAgo) return false;
            break;
          case 'month':
            const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
            if (transactionDate < monthAgo) return false;
            break;
          case 'custom':
            if (filters.customDateFrom && transactionDate < new Date(filters.customDateFrom)) return false;
            if (filters.customDateTo && transactionDate > new Date(filters.customDateTo)) return false;
            break;
        }
      }

      // Amount range filter
      if (filters.amountRange !== 'all') {
        const amount = Math.abs(transaction.amount);
        switch (filters.amountRange) {
          case 'small':
            if (amount >= 10000) return false; // Less than 10k MNT
            break;
          case 'medium':
            if (amount < 10000 || amount >= 100000) return false; // 10k-100k MNT
            break;
          case 'large':
            if (amount < 100000) return false; // 100k+ MNT
            break;
          case 'custom':
            if (filters.customAmountMin && amount < filters.customAmountMin) return false;
            if (filters.customAmountMax && amount > filters.customAmountMax) return false;
            break;
        }
      }

      // Text search filter
      if (filters.searchText) {
        const searchLower = filters.searchText.toLowerCase();
        if (!transaction.description.toLowerCase().includes(searchLower) &&
            !transaction.type.toLowerCase().includes(searchLower)) {
          return false;
        }
      }

      return true;
    });
  }, [transactions, filters]);

  // Paginated transactions
  const paginatedTransactions = useMemo(() => {
    const startIndex = (currentPage - 1) * itemsPerPage;
    return filteredTransactions.slice(startIndex, startIndex + itemsPerPage);
  }, [filteredTransactions, currentPage, itemsPerPage]);

  const totalPages = Math.ceil(filteredTransactions.length / itemsPerPage);

  const handleFilterChange = (key: keyof TransactionFilters, value: any) => {
    setFilters(prev => ({ ...prev, [key]: value }));
    setCurrentPage(1); // Reset to first page when filters change
  };

  const clearFilters = () => {
    setFilters(initialFilters);
    setCurrentPage(1);
  };

  return (
    <div>
      {/* Filter Controls */}
      <div className="mb-6">
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center space-x-4">
            <h4 className="text-lg font-medium text-gray-900">Transaction History</h4>
            <span className="text-sm text-gray-500">
              {filteredTransactions.length} of {transactions?.length || 0} transactions
            </span>
          </div>
          <div className="flex space-x-2">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded-md hover:bg-blue-200"
            >
              {showFilters ? 'Hide Filters' : 'Show Filters'}
            </button>
            {(filters.type !== 'all' || filters.dateRange !== 'all' || filters.amountRange !== 'all' || filters.searchText) && (
              <button
                onClick={clearFilters}
                className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200"
              >
                Clear Filters
              </button>
            )}
          </div>
        </div>

        {/* Search Bar */}
        <div className="mb-4">
          <input
            type="text"
            placeholder="Search by description or type..."
            value={filters.searchText}
            onChange={(e) => handleFilterChange('searchText', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* Advanced Filters */}
        {showFilters && (
          <div className="bg-gray-50 p-4 rounded-lg space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {/* Transaction Type Filter */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Transaction Type</label>
                <select
                  value={filters.type}
                  onChange={(e) => handleFilterChange('type', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="all">All Types</option>
                  <option value="game_buyin">Game Buy-in</option>
                  <option value="game_cashout">Game Cash-out</option>
                  <option value="tournament_buyin">Tournament Buy-in</option>
                  <option value="tournament_prize">Tournament Prize</option>
                  <option value="rake_collection">Rake Collection</option>
                </select>
              </div>

              {/* Date Range Filter */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Date Range</label>
                <select
                  value={filters.dateRange}
                  onChange={(e) => handleFilterChange('dateRange', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="all">All Time</option>
                  <option value="today">Today</option>
                  <option value="week">Last 7 Days</option>
                  <option value="month">Last 30 Days</option>
                  <option value="custom">Custom Range</option>
                </select>
              </div>

              {/* Amount Range Filter */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Amount Range</label>
                <select
                  value={filters.amountRange}
                  onChange={(e) => handleFilterChange('amountRange', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="all">All Amounts</option>
                  <option value="small">Small (&lt; 10K MNT)</option>
                  <option value="medium">Medium (10K - 100K MNT)</option>
                  <option value="large">Large (&gt; 100K MNT)</option>
                  <option value="custom">Custom Range</option>
                </select>
              </div>
            </div>

            {/* Custom Date Range */}
            {filters.dateRange === 'custom' && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">From Date</label>
                  <input
                    type="date"
                    value={filters.customDateFrom || ''}
                    onChange={(e) => handleFilterChange('customDateFrom', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">To Date</label>
                  <input
                    type="date"
                    value={filters.customDateTo || ''}
                    onChange={(e) => handleFilterChange('customDateTo', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>
            )}

            {/* Custom Amount Range */}
            {filters.amountRange === 'custom' && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Min Amount (MNT)</label>
                  <input
                    type="number"
                    value={filters.customAmountMin || ''}
                    onChange={(e) => handleFilterChange('customAmountMin', Number(e.target.value) || undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="0"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Max Amount (MNT)</label>
                  <input
                    type="number"
                    value={filters.customAmountMax || ''}
                    onChange={(e) => handleFilterChange('customAmountMax', Number(e.target.value) || undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="No limit"
                  />
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Transaction List */}
      {transactionLoading ? (
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-full mb-2"></div>
              <div className="h-3 bg-gray-200 rounded w-3/4"></div>
            </div>
          ))}
        </div>
      ) : transactionError ? (
        <div className="text-red-600">
          <p>Error loading transactions: {getErrorMessage(transactionError)}</p>
        </div>
      ) : filteredTransactions.length === 0 ? (
        <div className="text-gray-500 text-center py-8">
          {transactions?.length === 0 ? (
            <p>No transactions found</p>
          ) : (
            <p>No transactions match your filters</p>
          )}
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-2">Date</th>
                  <th className="text-left py-2">Type</th>
                  <th className="text-left py-2">Description</th>
                  <th className="text-right py-2">Amount</th>
                </tr>
              </thead>
              <tbody>
                {paginatedTransactions.map((transaction) => (
                  <tr key={transaction.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3">
                      {formatDateTime(transaction.created_at)}
                    </td>
                    <td className="py-3">
                      <span className={`inline-block px-2 py-1 rounded-full text-xs font-medium ${
                        transaction.type === 'game_buyin' || transaction.type === 'tournament_buyin'
                          ? 'bg-red-100 text-red-800'
                          : 'bg-green-100 text-green-800'
                      }`}>
                        {transaction.type.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="py-3 text-gray-600">
                      {transaction.description}
                    </td>
                    <td className={`py-3 text-right font-medium ${
                      transaction.amount >= 0 ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {transaction.amount >= 0 ? '+' : ''}{formatMNT(transaction.amount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-between items-center mt-4">
              <div className="text-sm text-gray-500">
                Showing {((currentPage - 1) * itemsPerPage) + 1} to {Math.min(currentPage * itemsPerPage, filteredTransactions.length)} of {filteredTransactions.length} transactions
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                  disabled={currentPage === 1}
                  className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <span className="px-3 py-1 text-sm text-gray-700">
                  Page {currentPage} of {totalPages}
                </span>
                <button
                  onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                  disabled={currentPage === totalPages}
                  className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}