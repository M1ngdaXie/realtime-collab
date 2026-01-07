import { IndexeddbPersistence } from "y-indexeddb";
import { Awareness } from "y-protocols/awareness";
import { WebsocketProvider } from "y-websocket";
import * as Y from "yjs";
import { documentsApi } from "../api/document";

const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws";

export interface YjsProviders {
  ydoc: Y.Doc;
  websocketProvider: WebsocketProvider;
  indexeddbProvider: IndexeddbPersistence;
  awareness: Awareness;
}

export interface YjsUser {
  id: string;
  name: string;
  avatar_url?: string;
}

export const createYjsDocument = async (
  documentId: string,
  user: YjsUser,
  token: string,
  shareToken?: string
): Promise<YjsProviders> => {
  // Create Yjs document
  const ydoc = new Y.Doc();

  // Load initial state from database BEFORE connecting providers
  try {
    const state = await documentsApi.getState(documentId);
    if (state && state.length > 0) {
      Y.applyUpdate(ydoc, state);
      console.log('✓ Loaded document state from database');
    }
  } catch {
    console.log('No existing state in database (new document)');
  }

  // IndexedDB persistence (offline support)
  const indexeddbProvider = new IndexeddbPersistence(documentId, ydoc);

  // WebSocket provider (real-time sync) with auth token
  const wsParams: { [key: string]: string } = shareToken
    ? { share: shareToken }
    : { token: token };
  const websocketProvider = new WebsocketProvider(
    WS_URL,
    documentId,
    ydoc,
    { params: wsParams }
  );

  // Awareness for cursors and presence
  const awareness = websocketProvider.awareness;

  return {
    ydoc,
    websocketProvider,
    indexeddbProvider,
    awareness,
  };
};

export const destroyYjsDocument = (providers: YjsProviders) => {
  providers.websocketProvider.destroy();
  providers.indexeddbProvider.destroy();
  providers.ydoc.destroy();
};

// Deterministic color generator based on user ID
export const getColorFromUserId = (userId: string | undefined): string => {
  // Default color if no userId
  if (!userId) {
    return "#718096"; // Gray for anonymous/undefined users
  }

  // Hash user ID to consistent color index
  const hash = userId.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  const colors = [
    "#FF6B6B",
    "#4ECDC4",
    "#45B7D1",
    "#FFA07A",
    "#98D8C8",
    "#F7DC6F",
    "#BB8FCE",
    "#85C1E2",
  ];
  return colors[hash % colors.length];
};
