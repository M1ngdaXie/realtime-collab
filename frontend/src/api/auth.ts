import type { User } from '../types/auth';
import { API_BASE_URL, authFetch } from './client';

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
