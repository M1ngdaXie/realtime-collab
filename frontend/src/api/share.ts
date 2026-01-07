import { authFetch, API_BASE_URL } from './client';
import type {
  DocumentShareWithUser,
  CreateShareRequest,
  ShareLink,
} from '../types/share';

export const shareApi = {
  // Create a share with a specific user
  createShare: async (
    documentId: string,
    request: CreateShareRequest
  ): Promise<DocumentShareWithUser> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${documentId}/shares`, {
      method: 'POST',
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error('Failed to create share');
    }

    return response.json();
  },

  // List all shares for a document
  listShares: async (documentId: string): Promise<DocumentShareWithUser[]> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${documentId}/shares`);

    if (!response.ok) {
      throw new Error('Failed to list shares');
    }

    const data = await response.json();
    return data.shares || [];
  },

  // Delete a share
  deleteShare: async (documentId: string, userId: string): Promise<void> => {
    const response = await authFetch(
      `${API_BASE_URL}/documents/${documentId}/shares/${userId}`,
      {
        method: 'DELETE',
      }
    );

    if (!response.ok) {
      throw new Error('Failed to delete share');
    }
  },

  // Create a public share link
  createShareLink: async (
    documentId: string,
    permission: 'view' | 'edit'
  ): Promise<ShareLink> => {
    const response = await authFetch(
      `${API_BASE_URL}/documents/${documentId}/share-link`,
      {
        method: 'POST',
        body: JSON.stringify({ permission }),
      }
    );

    if (!response.ok) {
      throw new Error('Failed to create share link');
    }

    return response.json();
  },

  // Get the current share link
  getShareLink: async (documentId: string): Promise<ShareLink | null> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${documentId}/share-link`);

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      throw new Error('Failed to get share link');
    }

    return response.json();
  },

  // Revoke the share link
  revokeShareLink: async (documentId: string): Promise<void> => {
    const response = await authFetch(
      `${API_BASE_URL}/documents/${documentId}/share-link`,
      {
        method: 'DELETE',
      }
    );

    if (!response.ok) {
      throw new Error('Failed to revoke share link');
    }
  },
};
