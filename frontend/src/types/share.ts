import type { User } from './auth';

export interface DocumentShare {
  id: string;
  document_id: string;
  user_id: string;
  permission: 'view' | 'edit';
  created_at: string;
  created_by?: string;
}

export interface DocumentShareWithUser extends DocumentShare {
  user: User;
}

export interface CreateShareRequest {
  user_email: string;
  permission: 'view' | 'edit';
}

export interface ShareLink {
  token: string;
  permission: 'view' | 'edit';
  created_at: string;
}
