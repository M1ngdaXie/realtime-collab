interface Config {
  apiUrl: string;
  wsUrl: string;
  environment: 'development' | 'production';
}

function loadConfig(): Config {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
  const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws';
  const environment = import.meta.env.MODE === 'production' ? 'production' : 'development';

  return {
    apiUrl,
    wsUrl,
    environment,
  };
}

export const config = loadConfig();

// Convenience exports
export const API_BASE_URL = config.apiUrl;
export const WS_URL = config.wsUrl;
export const isProduction = config.environment === 'production';
