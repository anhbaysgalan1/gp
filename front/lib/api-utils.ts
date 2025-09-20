/**
 * Utility functions for API operations
 * Includes helpers for formatting, validation, error handling, and common operations
 */

import { ApiError } from './api-client';
import {
  User,
  UserRole,
  TransactionType,
  CurrencyFormatOptions,
  FormErrors,
  ValidationError
} from '../types/api';

// ============= ERROR HANDLING UTILITIES =============

/**
 * Extract user-friendly error message from API error
 */
export function getErrorMessage(error: any): string {
  // Handle ApiError instances
  if (error instanceof ApiError) {
    return error.message;
  }

  // Handle response data from backend
  if (error?.response?.error) {
    return error.response.error;
  }

  if (error?.response?.message) {
    return error.response.message;
  }

  if (error?.response?.data?.error) {
    return error.response.data.error;
  }

  if (error?.response?.data?.message) {
    return error.response.data.message;
  }

  // Handle direct error objects
  if (typeof error === 'object' && error?.error) {
    return error.error;
  }

  if (error?.message) {
    return error.message;
  }

  // Handle string errors
  if (typeof error === 'string') {
    return error;
  }

  return 'An unexpected error occurred';
}

/**
 * Check if error is due to authentication failure
 */
export function isAuthError(error: any): boolean {
  if (error instanceof ApiError) {
    return error.status === 401;
  }
  return false;
}

/**
 * Check if error is due to insufficient permissions
 */
export function isPermissionError(error: any): boolean {
  if (error instanceof ApiError) {
    return error.status === 403;
  }
  return false;
}

/**
 * Parse validation errors from API response
 */
export function parseValidationErrors(error: any): FormErrors {
  const errors: FormErrors = {};

  if (error instanceof ApiError && error.response?.validation_errors) {
    error.response.validation_errors.forEach((validationError: ValidationError) => {
      errors[validationError.field] = validationError.message;
    });
  }

  return errors;
}

// ============= CURRENCY FORMATTING UTILITIES =============

/**
 * Format MNT currency amount
 */
export function formatMNT(
  amount: number,
  options: CurrencyFormatOptions = {}
): string {
  const {
    locale = 'mn-MN',
    minimumFractionDigits = 0,
    maximumFractionDigits = 0,
  } = options;

  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: 'MNT',
    minimumFractionDigits,
    maximumFractionDigits,
  }).format(amount);
}

/**
 * Format amount as compact currency (e.g., ₮1K, ₮1.5M)
 */
export function formatCompactMNT(amount: number): string {
  return new Intl.NumberFormat('mn-MN', {
    style: 'currency',
    currency: 'MNT',
    notation: 'compact',
    minimumFractionDigits: 0,
    maximumFractionDigits: 1,
  }).format(amount);
}

/**
 * Parse currency input string to number (removes currency symbols and formatting)
 */
export function parseCurrencyInput(value: string): number {
  const cleaned = value.replace(/[^\d.-]/g, '');
  const parsed = parseFloat(cleaned);
  return isNaN(parsed) ? 0 : parsed;
}

// ============= VALIDATION UTILITIES =============

/**
 * Validate email format
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

/**
 * Validate username format
 */
export function isValidUsername(username: string): boolean {
  // Username: 3-50 characters, alphanumeric and underscore only
  const usernameRegex = /^[a-zA-Z0-9_]{3,50}$/;
  return usernameRegex.test(username);
}

/**
 * Validate password strength
 */
export function isStrongPassword(password: string): boolean {
  // Password: At least 8 characters, must contain uppercase, lowercase, number, and special character
  const minLength = password.length >= 8;
  const hasUppercase = /[A-Z]/.test(password);
  const hasLowercase = /[a-z]/.test(password);
  const hasNumber = /\d/.test(password);
  const hasSpecial = /[!@#$%^&*(),.?":{}|<>]/.test(password);

  return minLength && hasUppercase && hasLowercase && hasNumber && hasSpecial;
}

/**
 * Get password strength score (0-4)
 */
export function getPasswordStrength(password: string): number {
  let score = 0;

  if (password.length >= 8) score++;
  if (/[A-Z]/.test(password)) score++;
  if (/[a-z]/.test(password)) score++;
  if (/\d/.test(password)) score++;
  if (/[!@#$%^&*(),.?":{}|<>]/.test(password)) score++;

  return score;
}

/**
 * Validate transfer amount
 */
export function isValidTransferAmount(amount: number, maxAmount?: number): boolean {
  return amount > 0 && (!maxAmount || amount <= maxAmount);
}

// ============= USER ROLE UTILITIES =============

/**
 * Check if user has specific role
 */
export function hasRole(user: User | null, role: UserRole): boolean {
  if (!user) return false;
  return user.role === role;
}

/**
 * Check if user has admin role
 */
export function isAdmin(user: User | null): boolean {
  return hasRole(user, 'admin');
}

/**
 * Check if user has moderator or admin role
 */
export function isModerator(user: User | null): boolean {
  if (!user) return false;
  return user.role === 'mod' || user.role === 'admin';
}

/**
 * Get user role display name
 */
export function getRoleDisplayName(role: UserRole): string {
  const roleNames: Record<UserRole, string> = {
    player: 'Player',
    mod: 'Moderator',
    admin: 'Administrator',
  };
  return roleNames[role];
}

/**
 * Get user role color class (for UI styling)
 */
export function getRoleColorClass(role: UserRole): string {
  const roleColors: Record<UserRole, string> = {
    player: 'text-blue-600',
    mod: 'text-yellow-600',
    admin: 'text-red-600',
  };
  return roleColors[role];
}

// ============= TRANSACTION UTILITIES =============

/**
 * Get transaction type display name
 */
export function getTransactionTypeDisplay(type: TransactionType): string {
  const typeNames: Record<TransactionType, string> = {
    game_buyin: 'Game Buy-in',
    game_cashout: 'Game Cash-out',
    tournament_buyin: 'Tournament Entry',
    tournament_prize: 'Tournament Prize',
    rake_collection: 'Rake Collection',
  };
  return typeNames[type];
}

/**
 * Get transaction type color class
 */
export function getTransactionTypeColor(type: TransactionType): string {
  const typeColors: Record<TransactionType, string> = {
    game_buyin: 'text-red-600',
    game_cashout: 'text-green-600',
    tournament_buyin: 'text-blue-600',
    tournament_prize: 'text-green-600',
    rake_collection: 'text-orange-600',
  };
  return typeColors[type];
}

/**
 * Check if transaction is a debit (money going out)
 */
export function isDebitTransaction(type: TransactionType): boolean {
  return type === 'game_buyin' || type === 'tournament_buyin' || type === 'rake_collection';
}

// ============= DATE/TIME UTILITIES =============

/**
 * Format date for display
 */
export function formatDate(dateString: string, options?: Intl.DateTimeFormatOptions): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...options,
  });
}

/**
 * Format date and time for display
 */
export function formatDateTime(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Get relative time (e.g., "2 hours ago")
 */
export function getRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (diffInSeconds < 60) {
    return 'Just now';
  }

  const diffInMinutes = Math.floor(diffInSeconds / 60);
  if (diffInMinutes < 60) {
    return `${diffInMinutes} minute${diffInMinutes > 1 ? 's' : ''} ago`;
  }

  const diffInHours = Math.floor(diffInMinutes / 60);
  if (diffInHours < 24) {
    return `${diffInHours} hour${diffInHours > 1 ? 's' : ''} ago`;
  }

  const diffInDays = Math.floor(diffInHours / 24);
  if (diffInDays < 7) {
    return `${diffInDays} day${diffInDays > 1 ? 's' : ''} ago`;
  }

  return formatDate(dateString);
}

// ============= LOCAL STORAGE UTILITIES =============

/**
 * Safe localStorage operations with error handling
 */
export const storage = {
  get: (key: string): string | null => {
    try {
      return typeof window !== 'undefined' ? localStorage.getItem(key) : null;
    } catch {
      return null;
    }
  },

  set: (key: string, value: string): void => {
    try {
      if (typeof window !== 'undefined') {
        localStorage.setItem(key, value);
      }
    } catch {
      // Silently fail if localStorage is not available
    }
  },

  remove: (key: string): void => {
    try {
      if (typeof window !== 'undefined') {
        localStorage.removeItem(key);
      }
    } catch {
      // Silently fail if localStorage is not available
    }
  },

  clear: (): void => {
    try {
      if (typeof window !== 'undefined') {
        localStorage.clear();
      }
    } catch {
      // Silently fail if localStorage is not available
    }
  },
};

// ============= URL UTILITIES =============

/**
 * Build query string from object
 */
export function buildQueryString(params: Record<string, any>): string {
  const searchParams = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value));
    }
  });

  const queryString = searchParams.toString();
  return queryString ? `?${queryString}` : '';
}

// ============= DEBOUNCE UTILITY =============

/**
 * Debounce function to limit API calls
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): T {
  let timeout: NodeJS.Timeout;

  return ((...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  }) as T;
}

// ============= RETRY UTILITY =============

/**
 * Retry async function with exponential backoff
 */
export async function retry<T>(
  fn: () => Promise<T>,
  maxAttempts: number = 3,
  baseDelay: number = 1000
): Promise<T> {
  let lastError: Error;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error('Unknown error');

      if (attempt === maxAttempts) {
        throw lastError;
      }

      // Exponential backoff
      const delay = baseDelay * Math.pow(2, attempt - 1);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError!;
}