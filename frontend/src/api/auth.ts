import type { User } from '../types/auth';
import { API_BASE_URL, authFetch } from './client';

export const authApi = {
  getCurrentUser: async (): Promise<User> => {
    const response = await authFetch(`${API_BASE_URL}/auth/me`);
    console.log("current user is " + response)
    if (!response.ok) {
      throw new Error('Failed to get current user');
    }

    return response.json();
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
