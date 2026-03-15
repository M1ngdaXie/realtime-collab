import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import type { ReactNode } from 'react';
import type { User, AuthContextType } from '../types/auth';
import { authApi } from '../api/auth';

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Initialize auth state on mount
  useEffect(() => {
    const initAuth = async () => {
      const storedToken = localStorage.getItem('auth_token');

      if (!storedToken) {
        setLoading(false);
        return;
      }

      try {
        setToken(storedToken);
        const currentUser = await authApi.getCurrentUser();
        setUser(currentUser);
        setError(null);
      } catch (err) {
        console.error('Failed to validate token:', err);
        localStorage.removeItem('auth_token');
        setToken(null);
        setUser(null);
        setError('Session expired');
      } finally {
        setLoading(false);
      }
    };

    initAuth();
  }, []);

  const login = useCallback(async (newToken: string) => {
    try {
      setLoading(true);
      setError(null);

      localStorage.setItem('auth_token', newToken);
      setToken(newToken);

      const currentUser = await authApi.getCurrentUser();
      setUser(currentUser);
    } catch (err) {
      console.error('Login failed:', err);
      localStorage.removeItem('auth_token');
      setToken(null);
      setUser(null);
      setError('Login failed');
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = () => {
    localStorage.removeItem('auth_token');
    setUser(null);
    setToken(null);
    setError(null);

    // Call backend logout endpoint (fire and forget)
    authApi.logout().catch(console.error);

    // Redirect to login
    window.location.href = '/login';
  };

  const refreshUser = async () => {
    if (!token) return;

    try {
      const currentUser = await authApi.getCurrentUser();
      setUser(currentUser);
      setError(null);
    } catch (err) {
      console.error('Failed to refresh user:', err);
      setError('Failed to refresh user');
    }
  };

  const value: AuthContextType = {
    user,
    token,
    loading,
    error,
    login,
    logout,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
