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

export const createYjsDocument = async (documentId: string): Promise<YjsProviders> => {
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

  // WebSocket provider (real-time sync)
  const websocketProvider = new WebsocketProvider(WS_URL, documentId, ydoc);

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

// Random color generator for users
export const getRandomColor = () => {
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
  return colors[Math.floor(Math.random() * colors.length)];
};

// Random name generator
export const getRandomName = () => {
  const adjectives = ["Happy", "Clever", "Brave", "Swift", "Kind"];
  const animals = ["Panda", "Fox", "Wolf", "Bear", "Eagle"];
  return `${adjectives[Math.floor(Math.random() * adjectives.length)]} ${
    animals[Math.floor(Math.random() * animals.length)]
  }`;
};
