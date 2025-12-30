const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

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

export const documentsApi = {
  // List all documents
  list: async (): Promise<{ documents: DocumentType[]; total: number }> => {
    const response = await fetch(`${API_BASE_URL}/documents`);
    if (!response.ok) throw new Error("Failed to fetch documents");
    return response.json();
  },

  // Get a single document
  get: async (id: string): Promise<DocumentType> => {
    const response = await fetch(`${API_BASE_URL}/documents/${id}`);
    if (!response.ok) throw new Error("Failed to fetch document");
    return response.json();
  },

  // Create a new document
  create: async (data: CreateDocumentRequest): Promise<DocumentType> => {
    const response = await fetch(`${API_BASE_URL}/documents`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error("Failed to create document");
    return response.json();
  },

  // Delete a document
  delete: async (id: string): Promise<void> => {
    const response = await fetch(`${API_BASE_URL}/documents/${id}`, {
      method: "DELETE",
    });
    if (!response.ok) throw new Error("Failed to delete document");
  },

  // Get document Yjs state
  getState: async (id: string): Promise<Uint8Array> => {
    const response = await fetch(`${API_BASE_URL}/documents/${id}/state`);
    if (!response.ok) throw new Error("Failed to fetch document state");
    const arrayBuffer = await response.arrayBuffer();
    return new Uint8Array(arrayBuffer);
  },

  // Update document Yjs state
  updateState: async (id: string, state: Uint8Array): Promise<void> => {
    // Create a new ArrayBuffer copy to ensure compatibility
    const buffer = new ArrayBuffer(state.byteLength);
    new Uint8Array(buffer).set(state);

    const response = await fetch(`${API_BASE_URL}/documents/${id}/state`, {
      method: "PUT",
      headers: { "Content-Type": "application/octet-stream" },
      body: buffer,
    });
    if (!response.ok) throw new Error("Failed to update document state");
  },
};