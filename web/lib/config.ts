/**
 * Frontend configuration and environment variables
 */

export const config = {
  // API Configuration
  api: {
    baseUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    timeout: 30000, // 30 seconds
    retries: 3,
  },

  // WebSocket Configuration
  websocket: {
    url: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080',
    reconnectInterval: 5000, // 5 seconds
    maxReconnectAttempts: 5,
  },

  // Application Configuration
  app: {
    name: 'Go Poker Platform',
    version: process.env.NEXT_PUBLIC_APP_VERSION || '1.0.0',
    environment: process.env.NODE_ENV || 'development',
  },

  // Feature Flags
  features: {
    registration: process.env.NEXT_PUBLIC_ENABLE_REGISTRATION !== 'false',
    emailVerification: process.env.NEXT_PUBLIC_REQUIRE_EMAIL_VERIFICATION !== 'false',
    socialLogin: process.env.NEXT_PUBLIC_ENABLE_SOCIAL_LOGIN === 'true',
    analytics: process.env.NEXT_PUBLIC_ENABLE_ANALYTICS === 'true',
  },

  // UI Configuration
  ui: {
    theme: {
      primary: '#2563eb', // blue-600
      secondary: '#64748b', // slate-500
      success: '#059669', // emerald-600
      warning: '#d97706', // amber-600
      error: '#dc2626', // red-600
    },
    currency: {
      code: 'MNT',
      symbol: '₮',
      locale: 'mn-MN',
    },
  },

  // Game Configuration
  game: {
    defaultBuyIn: 10000, // 10,000 MNT
    maxBuyIn: 1000000, // 1,000,000 MNT
    minBuyIn: 1000, // 1,000 MNT
    maxPlayers: 8,
    defaultGameSpeed: 'normal', // 'slow', 'normal', 'fast'
  },

  // Security Configuration
  security: {
    tokenStorageKey: 'auth_token',
    sessionTimeout: 24 * 60 * 60 * 1000, // 24 hours
    maxLoginAttempts: 5,
    lockoutDuration: 15 * 60 * 1000, // 15 minutes
  },

  // Development Configuration
  development: {
    enableLogging: process.env.NODE_ENV === 'development',
    enableDebugPanel: process.env.NODE_ENV === 'development',
    mockData: process.env.NEXT_PUBLIC_USE_MOCK_DATA === 'true',
  },
} as const;

// Environment validation
export function validateEnvironment(): void {
  const requiredEnvVars = [
    'NEXT_PUBLIC_API_URL',
    'NEXT_PUBLIC_WS_URL',
  ];

  const missingVars = requiredEnvVars.filter(
    (varName) => !process.env[varName]
  );

  if (missingVars.length > 0) {
    console.warn(
      `Missing environment variables: ${missingVars.join(', ')}`
    );
  }
}

// Initialize environment validation in development
if (config.development.enableLogging) {
  validateEnvironment();
}

export default config;