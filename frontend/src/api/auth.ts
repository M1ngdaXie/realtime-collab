import type { User } from '../types/auth';
import { API_BASE_URL, authFetch } from './client';

export async function guestLogin(): Promise<string> {
  const res = await fetch(`${API_BASE_URL}/auth/guest`, { method: 'POST' });
  if (!res.ok) {
    throw new Error('Failed to create guest session');
  }
  const data = await res.json();
  return data.token;
}

export const authApi = {
  getCurrentUser: async (): Promise<User> => {
    const response = await authFetch(`${API_BASE_URL}/auth/me`);
    if (!response.ok) {
      throw new Error('Failed to get current user');
    }

    const data = await response.json();
    console.log("current user data:", data);

    // Backend returns { user: {...} }, extract the user object
    return data.user;
  },

  logout: async (): Promise<void> => {
    const response = await authFetch(`${API_BASE_URL}/auth/logout`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error('Failed to logout');
    }
  },
};
