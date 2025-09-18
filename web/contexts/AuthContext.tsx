/**
 * Authentication Context Provider
 * Provides authentication state and actions throughout the application
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { useRouter } from 'next/router';
import { useAuth } from '../hooks/useApi';
import { User, LoginRequest, CreateUserRequest } from '../types/api';
import { getErrorMessage, isAuthError } from '../lib/api-utils';

interface AuthContextType {
  // State
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  error: Error | null;

  // Actions
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: CreateUserRequest) => Promise<void>;
  logout: () => Promise<void>;
  verifyEmail: (token: string) => Promise<void>;
  clearError: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const router = useRouter();

  // Authentication state with proper loading management
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isLoading, setIsLoading] = useState(true); // Start as loading
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Initialize authentication state on app load
  useEffect(() => {
    const initializeAuth = async () => {
      try {
        // Check if we have a stored token
        const token = localStorage.getItem('auth_token');
        if (token) {
          // Validate token by fetching user profile
          const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/user/me`, {
            headers: {
              'Authorization': `Bearer ${token}`,
              'Content-Type': 'application/json'
            },
          });

          if (response.ok) {
            const userData = await response.json();
            setUser(userData);
            setIsAuthenticated(true);
          } else {
            // Token is invalid, clear it
            localStorage.removeItem('auth_token');
          }
        }
      } catch (error) {
        console.error('Auth initialization error:', error);
        // Clear invalid token
        localStorage.removeItem('auth_token');
      } finally {
        setIsLoading(false);
      }
    };

    initializeAuth();
  }, []);

  // Clear error when route changes
  useEffect(() => {
    const handleRouteChange = () => {
      setError(null);
    };

    router.events.on('routeChangeStart', handleRouteChange);
    return () => {
      router.events.off('routeChangeStart', handleRouteChange);
    };
  }, [router.events]);

  const login = async (credentials: LoginRequest) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentials),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Login failed' }));
        throw new Error(errorData.message || 'Login failed');
      }

      const data = await response.json();

      // Store token in localStorage
      localStorage.setItem('auth_token', data.token);

      // Update state
      setUser(data.user);
      setIsAuthenticated(true);

      // Redirect to dashboard or intended page after successful login
      const returnTo = router.query.returnTo as string;
      const redirectTo = returnTo || '/dashboard';

      await router.push(redirectTo);
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Login failed');
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const register = async (userData: CreateUserRequest) => {
    setIsLoading(true);
    setError(null);

    try {
      // Use API client directly for now - will implement hooks later
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(userData),
      });

      if (!response.ok) {
        throw new Error('Registration failed');
      }

      // Redirect to verification pending page
      await router.push('/auth/verify-email-pending');
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Registration failed');
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    setIsLoading(true);
    setError(null);

    try {
      // Clear token from localStorage
      localStorage.removeItem('auth_token');

      // Update state
      setUser(null);
      setIsAuthenticated(false);

      // Redirect to home page after logout
      await router.push('/');
    } catch (err) {
      // Log error but don't prevent logout UI updates
      console.error('Logout error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const verifyEmail = async (token: string) => {
    setIsLoading(true);
    setError(null);

    try {
      // Use API client directly for now - will implement hooks later
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/verify-email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token }),
      });

      if (!response.ok) {
        throw new Error('Email verification failed');
      }

      // Redirect to login with success message
      await router.push('/auth/login?verified=true');
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Email verification failed');
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const clearError = () => {
    setError(null);
  };

  const refreshUser = async () => {
    // Will implement later
  };

  const value: AuthContextType = {
    // State
    user,
    isLoading,
    isAuthenticated,
    error,

    // Actions
    login,
    register,
    logout,
    verifyEmail,
    clearError,
    refreshUser,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

/**
 * Hook to use authentication context
 */
export function useAuthContext(): AuthContextType {
  const context = useContext(AuthContext);

  if (context === undefined) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }

  return context;
}

/**
 * Higher-order component to protect routes that require authentication
 */
export interface WithAuthOptions {
  redirectTo?: string;
  roles?: string[];
}

export function withAuth<P extends object>(
  Component: React.ComponentType<P>,
  options: WithAuthOptions = {}
) {
  return function AuthenticatedComponent(props: P) {
    const { user, isLoading, isAuthenticated } = useAuthContext();
    const router = useRouter();
    const { redirectTo = '/auth/login', roles } = options;

    useEffect(() => {
      if (!isLoading && !isAuthenticated) {
        const returnTo = router.asPath;
        router.push(`${redirectTo}?returnTo=${encodeURIComponent(returnTo)}`);
      }
    }, [isLoading, isAuthenticated, router]);

    useEffect(() => {
      if (user && roles && !roles.includes(user.role)) {
        // User doesn't have required role, redirect to unauthorized page
        router.push('/unauthorized');
      }
    }, [user, roles, router]);

    // Show loading state while checking authentication
    if (isLoading) {
      return (
        <div className="min-h-screen flex items-center justify-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
        </div>
      );
    }

    // Don't render component if not authenticated
    if (!isAuthenticated) {
      return null;
    }

    // Don't render component if user doesn't have required role
    if (user && roles && !roles.includes(user.role)) {
      return null;
    }

    return <Component {...props} />;
  };
}

/**
 * Hook to check if user has specific permissions
 */
export function usePermissions() {
  const { user } = useAuthContext();

  const hasRole = (role: string): boolean => {
    return user?.role === role;
  };

  const hasAnyRole = (roles: string[]): boolean => {
    return user ? roles.includes(user.role) : false;
  };

  const isAdmin = (): boolean => {
    return hasRole('admin');
  };

  const isModerator = (): boolean => {
    return hasAnyRole(['mod', 'admin']);
  };

  const canAccessAdmin = (): boolean => {
    return isAdmin();
  };

  const canModerate = (): boolean => {
    return isModerator();
  };

  return {
    hasRole,
    hasAnyRole,
    isAdmin,
    isModerator,
    canAccessAdmin,
    canModerate,
  };
}