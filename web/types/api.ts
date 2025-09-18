/**
 * TypeScript type definitions for API requests and responses
 * These match the Go backend models and API contracts
 */

// ============= USER TYPES =============

export interface User {
  id: string;
  email: string;
  username: string;
  role: UserRole;
  is_verified: boolean;
  avatar_url?: string;
  created_at: string;
  updated_at: string;
}

export type UserRole = 'player' | 'mod' | 'admin';

// ============= AUTHENTICATION TYPES =============

export interface CreateUserRequest {
  email: string;
  username: string;
  password: string;
}

export interface LoginRequest {
  email_or_username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface RegisterResponse {
  message: string;
  user_id: string;
}

// ============= BALANCE TYPES =============

export interface UserBalance {
  main_balance: number;    // Balance in MNT (Mongolian Tugrik)
  game_balance: number;    // Game account balance in MNT
  total_balance: number;   // Total balance (main + game) in MNT
}

export interface TransferRequest {
  amount: number;
  session_id: string;
}

export interface TransferResponse {
  message: string;
  transaction_id: string;
  amount: number;
  session_id: string;
}

// ============= TRANSACTION TYPES =============

export interface Transaction {
  id: string;
  user_id: string;
  type: TransactionType;
  amount: number;
  currency: string;
  created_at: string;
  description: string;
}

export type TransactionType =
  | 'game_buyin'
  | 'game_cashout'
  | 'tournament_buyin'
  | 'tournament_prize'
  | 'rake_collection';

export interface TransactionHistory {
  transactions: Transaction[];
  pagination: {
    limit: number;
    offset: number;
    total: number;
  };
}

// ============= ADMIN TYPES =============

export interface AdminUserListResponse {
  users: User[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    pages: number;
  };
}

export interface UpdateUserRoleRequest {
  role: UserRole;
}

// ============= ERROR TYPES =============

export interface ApiErrorResponse {
  error: string;
  message?: string;
  details?: any;
}

// ============= GENERAL RESPONSE TYPES =============

export interface MessageResponse {
  message: string;
}

export interface HealthCheckResponse {
  status: string;
  timestamp: string;
}

// ============= FORM VALIDATION TYPES =============

export interface FormErrors {
  [key: string]: string | undefined;
}

export interface ValidationError {
  field: string;
  message: string;
}

// ============= PAGINATION TYPES =============

export interface PaginationParams {
  page?: number;
  limit?: number;
  offset?: number;
}

export interface PaginationResponse {
  page: number;
  limit: number;
  total: number;
  pages: number;
}

// ============= API CLIENT TYPES =============

export interface ApiClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
}

// ============= CURRENCY UTILITY TYPES =============

export interface CurrencyFormatOptions {
  locale?: string;
  currency?: string;
  minimumFractionDigits?: number;
  maximumFractionDigits?: number;
}

// Helper type for API endpoints that require authentication
export type AuthenticatedEndpoint<T> = T & {
  requiresAuth: true;
};

// Helper type for API endpoints that require specific roles
export type RoleRestrictedEndpoint<T> = T & {
  requiredRoles: UserRole[];
};

// ============= TABLE TYPES =============

export interface PokerTable {
  id: string;
  name: string;
  table_type: 'cash' | 'tournament';
  game_type: 'texas_holdem' | 'omaha' | 'stud';
  max_players: number;
  min_buy_in: number;
  max_buy_in: number;
  small_blind: number;
  big_blind: number;
  is_private: boolean;
  status: 'waiting' | 'active' | 'full' | 'closed';
  current_players: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTableRequest {
  name: string;
  table_type: 'cash' | 'tournament';
  game_type?: 'texas_holdem' | 'omaha' | 'stud';
  max_players?: number;
  min_buy_in: number;
  max_buy_in: number;
  small_blind: number;
  big_blind: number;
  is_private?: boolean;
  password?: string;
}

export interface UpdateTableRequest {
  name?: string;
  is_private?: boolean;
  password?: string;
  max_buy_in?: number;
  min_buy_in?: number;
  small_blind?: number;
  big_blind?: number;
}

export interface JoinTableRequest {
  buy_in_amount: number;
  password?: string;
}

export interface JoinTableResponse {
  message: string;
  table_id: string;
  user_id: string;
  buy_in_amount: number;
  session_id: string;
}

export interface TableListResponse {
  tables: PokerTable[];
  pagination: {
    limit: number;
    offset: number;
    total: number;
  };
}

// ============= WEBSOCKET GAME TYPES (extending existing) =============

// These extend the existing game interfaces but add API-related fields
export interface GameSession {
  id: string;
  table_name: string;
  user_id: string;
  buy_in_amount: number;
  current_balance: number;
  started_at: string;
  ended_at?: string;
  status: 'active' | 'completed' | 'abandoned';
}

export interface SessionInfo {
  user_id: string;
  session_id?: string;
  seat_number?: number;
  is_seated: boolean;
  has_session: boolean;
}

export interface TableInfo {
  name: string;
  max_players: number;
  current_players: number;
  max_buy_in: number;
  min_buy_in: number;
  small_blind: number;
  big_blind: number;
  status: 'waiting' | 'active' | 'paused';
}

// ============= REAL-TIME UPDATES TYPES =============

export interface BalanceUpdateEvent {
  type: 'balance_update';
  user_id: string;
  old_balance: UserBalance;
  new_balance: UserBalance;
  transaction_id?: string;
}

export interface TransactionEvent {
  type: 'transaction';
  user_id: string;
  transaction: Transaction;
}

// ============= API HOOKS TYPES =============

export interface UseApiOptions {
  suspense?: boolean;
  revalidateOnFocus?: boolean;
  revalidateOnReconnect?: boolean;
  refreshInterval?: number;
}

export interface UseApiResult<T> {
  data?: T;
  error?: Error;
  isLoading: boolean;
  isValidating: boolean;
  mutate: (data?: T) => Promise<void>;
}

// ============= ENVIRONMENT TYPES =============

export interface Environment {
  API_URL: string;
  WS_URL: string;
  NODE_ENV: 'development' | 'production' | 'test';
}