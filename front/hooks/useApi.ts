/**
 * Custom React hooks for API calls using SWR
 * Provides a convenient interface for data fetching with caching, revalidation, and error handling
 */

import useSWR, { SWRConfiguration, SWRResponse } from 'swr';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { apiClient } from '../lib/api-client';
import {
  User,
  UserBalance,
  TransactionHistory,
  AdminUserListResponse,
  CreateUserRequest,
  LoginRequest,
  TransferRequest,
  UseApiResult,
} from '../types/api';

// ============= BASE HOOKS =============

/**
 * Base SWR configuration
 */
const defaultConfig: SWRConfiguration = {
  revalidateOnFocus: false,
  revalidateOnReconnect: true,
  refreshInterval: 0,
  errorRetryCount: 3,
  errorRetryInterval: 1000,
};

/**
 * Custom SWR hook with better TypeScript support
 */
function useApiSWR<T>(
  key: string | null,
  fetcher: () => Promise<T>,
  config?: SWRConfiguration
): UseApiResult<T> {
  const { data, error, isValidating, mutate: swrMutate } = useSWR(
    key,
    fetcher,
    {
      ...defaultConfig,
      ...config,
    }
  );

  const mutate = useCallback(async (data?: T) => {
    if (data !== undefined) {
      await swrMutate(data, false);
    } else {
      await swrMutate();
    }
  }, [swrMutate]);

  return {
    data,
    error,
    isLoading: !data && !error,
    isValidating,
    mutate,
  };
}

// ============= AUTHENTICATION HOOKS =============

/**
 * Hook for managing user authentication state
 */
export function useAuth() {
  const { data: user, error, isLoading, mutate } = useApiSWR<User>(
    apiClient.isAuthenticated() ? '/api/user/profile' : null,
    () => apiClient.getCurrentUser(),
    {
      revalidateOnFocus: true,
      refreshInterval: 5 * 60 * 1000, // Refresh every 5 minutes
    }
  );

  const login = useCallback(async (credentials: LoginRequest) => {
    try {
      const response = await apiClient.login(credentials);
      await mutate(); // Refresh user data
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const register = useCallback(async (userData: CreateUserRequest) => {
    try {
      const response = await apiClient.register(userData);
      return response;
    } catch (error) {
      throw error;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient.logout();
      await mutate(); // Clear user data
    } catch (error) {
      // Still clear user data even if logout request fails
      await mutate();
      throw error;
    }
  }, [mutate]);

  const verifyEmail = useCallback(async (token: string) => {
    try {
      const response = await apiClient.verifyEmail(token);
      await mutate(); // Refresh user data to update verification status
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  return {
    user,
    error,
    isLoading,
    isAuthenticated: !!user && !error,
    login,
    register,
    logout,
    verifyEmail,
    mutate,
  };
}

// ============= USER PROFILE HOOKS =============

/**
 * Hook for user profile management
 */
export function useUserProfile() {
  const { data: user, error, isLoading, mutate } = useApiSWR<User>(
    apiClient.isAuthenticated() ? '/api/user/profile' : null,
    () => apiClient.getCurrentUser()
  );

  const updateProfile = useCallback(async (updates: Partial<User>) => {
    try {
      const response = await apiClient.updateProfile(updates);
      await mutate(); // Refresh user data from server
      return response;
    } catch (error) {
      await mutate(); // Refresh on error
      throw error;
    }
  }, [mutate]);

  const changePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    try {
      const response = await apiClient.changePassword(currentPassword, newPassword);
      return response;
    } catch (error) {
      throw error;
    }
  }, []);

  return {
    user,
    error,
    isLoading,
    updateProfile,
    changePassword,
    mutate,
  };
}

// ============= BALANCE HOOKS =============

/**
 * Hook for user balance management
 */
export function useBalance() {
  const { data: balance, error, isLoading, mutate } = useApiSWR<UserBalance>(
    apiClient.isAuthenticated() ? '/api/balance' : null,
    () => apiClient.getBalance(),
    {
      refreshInterval: 30 * 1000, // Refresh every 30 seconds
      revalidateOnFocus: true,
    }
  );

  // Listen for real-time balance updates from WebSocket
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handleBalanceUpdate = (event: CustomEvent) => {
      const newBalance = event.detail;
      // Update SWR cache with new balance data
      mutate(newBalance); // Update with new balance
    };

    window.addEventListener('balance-update', handleBalanceUpdate as EventListener);

    return () => {
      window.removeEventListener('balance-update', handleBalanceUpdate as EventListener);
    };
  }, [mutate]);

  const transferToGame = useCallback(async (amount: number, sessionId: string) => {
    try {
      const response = await apiClient.transferToGame(amount, sessionId);
      await mutate(); // Refresh balance after transfer
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const transferFromGame = useCallback(async (amount: number, sessionId: string) => {
    try {
      const response = await apiClient.transferFromGame(amount, sessionId);
      await mutate(); // Refresh balance after transfer
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const withdraw = useCallback(async (amount: number) => {
    try {
      const response = await apiClient.withdraw(amount);
      await mutate(); // Refresh balance after withdrawal
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const requestBalanceUpdate = useCallback(() => {
    // Send get-balance request via WebSocket if available
    if (typeof window !== 'undefined' && window.WebSocket) {
      window.dispatchEvent(new CustomEvent('request-balance', {}));
    }
  }, []);

  return {
    balance,
    error,
    isLoading,
    transferToGame,
    transferFromGame,
    withdraw,
    requestBalanceUpdate,
    mutate,
  };
}

// ============= TRANSACTION HISTORY HOOKS =============

/**
 * Hook for transaction history
 */
export function useTransactionHistory(limit = 10, offset = 0) {
  const key = apiClient.isAuthenticated()
    ? `/api/balance/transactions?limit=${limit}&offset=${offset}`
    : null;

  const { data: history, error, isLoading, mutate } = useApiSWR<TransactionHistory>(
    key,
    () => apiClient.getTransactionHistory(limit, offset)
  );

  return {
    transactions: history?.transactions || [],
    pagination: history?.pagination,
    error,
    isLoading,
    mutate,
  };
}

// ============= ADMIN HOOKS =============

/**
 * Hook for admin user management
 */
export function useAdminUsers(page = 1, limit = 20) {
  const key = apiClient.isAuthenticated()
    ? `/api/admin/users?page=${page}&limit=${limit}`
    : null;

  const { data, error, isLoading, mutate } = useApiSWR<AdminUserListResponse>(
    key,
    () => apiClient.getAllUsers(page, limit)
  );

  const updateUserRole = useCallback(async (userId: string, role: string) => {
    try {
      const response = await apiClient.updateUserRole(userId, role);
      await mutate(); // Refresh user list
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const deleteUser = useCallback(async (userId: string) => {
    try {
      const response = await apiClient.deleteUser(userId);
      await mutate(); // Refresh user list
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const depositMoney = useCallback(async (userId: string, amount: number) => {
    try {
      const response = await apiClient.depositMoney(userId, amount);
      await mutate(); // Refresh user list to show updated balances if needed
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  const withdrawMoney = useCallback(async (userId: string, amount: number) => {
    try {
      const response = await apiClient.withdrawMoney(userId, amount);
      await mutate(); // Refresh user list to show updated balances if needed
      return response;
    } catch (error) {
      throw error;
    }
  }, [mutate]);

  return {
    users: data?.users || [],
    pagination: data?.pagination,
    error,
    isLoading,
    updateUserRole,
    deleteUser,
    depositMoney,
    withdrawMoney,
    mutate,
  };
}

// ============= UTILITY HOOKS =============

/**
 * Hook for API health check
 */
export function useHealthCheck() {
  const { data, error, isLoading } = useApiSWR(
    '/health',
    () => apiClient.healthCheck(),
    {
      refreshInterval: 60 * 1000, // Check every minute
      revalidateOnFocus: false,
    }
  );

  return {
    isHealthy: !!data && data.status === 'ok',
    healthData: data,
    error,
    isLoading,
  };
}

// ============= MUTATION HOOKS =============

/**
 * Hook for API mutations with loading state
 */
export function useApiMutation<TData, TVariables = any>(
  mutationFn: (variables: TVariables) => Promise<TData>
) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const mutate = useCallback(async (variables: TVariables): Promise<TData> => {
    setIsLoading(true);
    setError(null);

    try {
      const result = await mutationFn(variables);
      return result;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error');
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [mutationFn]);

  return {
    mutate,
    isLoading,
    error,
  };
}

// Export commonly used mutation hooks
export function useLoginMutation() {
  return useApiMutation((credentials: LoginRequest) =>
    apiClient.login(credentials)
  );
}

export function useRegisterMutation() {
  return useApiMutation((userData: CreateUserRequest) =>
    apiClient.register(userData)
  );
}

export function useTransferMutation() {
  return useApiMutation(({ type, amount, sessionId }: {
    type: 'to-game' | 'from-game';
    amount: number;
    sessionId: string;
  }) => {
    return type === 'to-game'
      ? apiClient.transferToGame(amount, sessionId)
      : apiClient.transferFromGame(amount, sessionId);
  });
}

export function useWithdrawMutation() {
  return useApiMutation((amount: number) =>
    apiClient.withdraw(amount)
  );
}