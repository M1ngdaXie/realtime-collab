import { authFetch, API_BASE_URL } from './client';

export type DocumentType = {
  id: string;
  name: string;
  type: "editor" | "kanban";
  created_at: string;
  updated_at: string;
}

export type CreateDocumentRequest = {
  name: string;
  type: "editor" | "kanban";
}

export type PermissionResponse = {
  permission: "view" | "edit";
  role: "owner" | "editor" | "viewer";
}

export const documentsApi = {
  // List all documents
  list: async (): Promise<{ documents: DocumentType[]; total: number }> => {
    const response = await authFetch(`${API_BASE_URL}/documents`);
    if (!response.ok) throw new Error("Failed to fetch documents");
    return response.json();
  },

  // Get a single document
  get: async (id: string): Promise<DocumentType> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${id}`);
    if (!response.ok) throw new Error("Failed to fetch document");
    return response.json();
  },

  // Create a new document
  create: async (data: CreateDocumentRequest): Promise<DocumentType> => {
    const response = await authFetch(`${API_BASE_URL}/documents`, {
      method: "POST",
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error("Failed to create document");
    return response.json();
  },

  // Delete a document
  delete: async (id: string): Promise<void> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${id}`, {
      method: "DELETE",
    });
    if (!response.ok) throw new Error("Failed to delete document");
  },

  // Get document Yjs state
  getState: async (id: string): Promise<Uint8Array> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${id}/state`);
    if (!response.ok) throw new Error("Failed to fetch document state");
    const arrayBuffer = await response.arrayBuffer();
    return new Uint8Array(arrayBuffer);
  },

  // Update document Yjs state
  updateState: async (id: string, state: Uint8Array): Promise<void> => {
    // Create a new ArrayBuffer copy to ensure compatibility
    const buffer = new ArrayBuffer(state.byteLength);
    new Uint8Array(buffer).set(state);

    const response = await authFetch(`${API_BASE_URL}/documents/${id}/state`, {
      method: "PUT",
      headers: { "Content-Type": "application/octet-stream" },
      body: buffer,
    });
    if (!response.ok) throw new Error("Failed to update document state");
  },

  // Get user's permission for a document
  getPermission: async (id: string, shareToken?: string): Promise<PermissionResponse> => {
    const url = shareToken
      ? `${API_BASE_URL}/documents/${id}/permission?share=${shareToken}`
      : `${API_BASE_URL}/documents/${id}/permission`;

    const response = await authFetch(url);
    if (!response.ok) throw new Error("Failed to fetch document permission");
    return response.json();
  },
};

// Version History Types
export type DocumentVersion = {
  id: string;
  document_id: string;
  text_preview: string | null;
  version_number: number;
  created_by: string | null;
  version_label: string | null;
  is_auto_generated: boolean;
  created_at: string;
  author?: {
    id: string;
    email: string;
    name: string;
    avatar_url?: string;
  };
};

export type VersionListResponse = {
  versions: DocumentVersion[];
  total: number;
};

// Version History API
export const versionsApi = {
  // Create manual snapshot
  create: async (
    documentId: string,
    yjsSnapshot: Uint8Array,
    textPreview: string,
    versionLabel?: string
  ): Promise<DocumentVersion> => {
    const formData = new FormData();
    // Create a copy of the buffer to ensure compatibility
    const buffer = new ArrayBuffer(yjsSnapshot.byteLength);
    new Uint8Array(buffer).set(yjsSnapshot);
    formData.append('yjs_snapshot', new Blob([buffer]));
    formData.append('text_preview', textPreview);
    if (versionLabel) {
      formData.append('version_label', versionLabel);
    }
    const response = await authFetch(`${API_BASE_URL}/documents/${documentId}/versions`, {
      method: 'POST',
      body: formData,
    });
    if (!response.ok) throw new Error('Failed to create version');
    return response.json();
  },

  // List versions (paginated)
  list: async (
    documentId: string,
    limit: number = 50,
    offset: number = 0
  ): Promise<VersionListResponse> => {
    const response = await authFetch(
      `${API_BASE_URL}/documents/${documentId}/versions?limit=${limit}&offset=${offset}`
    );
    if (!response.ok) throw new Error('Failed to fetch versions');
    return response.json();
  },

  // Get version snapshot (binary)
  getSnapshot: async (documentId: string, versionId: string): Promise<Uint8Array> => {
    const response = await authFetch(
      `${API_BASE_URL}/documents/${documentId}/versions/${versionId}/snapshot`
    );
    if (!response.ok) throw new Error('Failed to fetch snapshot');
    return new Uint8Array(await response.arrayBuffer());
  },

  // Restore version
  restore: async (documentId: string, versionId: string): Promise<{ message: string }> => {
    const response = await authFetch(`${API_BASE_URL}/documents/${documentId}/restore`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version_id: versionId }),
    });
    if (!response.ok) throw new Error('Failed to restore version');
    return response.json();
  },
};