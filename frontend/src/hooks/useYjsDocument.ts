import { useEffect, useState } from "react";
import {
    createYjsDocument,
    destroyYjsDocument,
    getRandomColor,
    getRandomName,
    type YjsProviders,
} from "../lib/yjs";
import { useAutoSave } from "./useAutoSave";

export const useYjsDocument = (documentId: string) => {
  const [providers, setProviders] = useState<YjsProviders | null>(null);
  const [synced, setSynced] = useState(false);

  // Enable auto-save when providers are ready
  useAutoSave(documentId, providers?.ydoc || null);

  useEffect(() => {
    let mounted = true;
    let currentProviders: YjsProviders | null = null;

    // Create Yjs document and providers
    const initializeDocument = async () => {
      const yjsProviders = await createYjsDocument(documentId);
      currentProviders = yjsProviders;

      if (!mounted) {
        destroyYjsDocument(yjsProviders);
        return;
      }

      // Set user info for awareness
      yjsProviders.awareness.setLocalStateField("user", {
        name: getRandomName(),
        color: getRandomColor(),
      });

      // Listen for sync status
      yjsProviders.indexeddbProvider.on("synced", () => {
        console.log("IndexedDB synced");
        setSynced(true);
      });

      yjsProviders.websocketProvider.on("status", (event: { status: string }) => {
        console.log("WebSocket status:", event.status);
      });

      setProviders(yjsProviders);
    };

    initializeDocument();

    // Cleanup on unmount
    return () => {
      mounted = false;
      if (currentProviders) {
        destroyYjsDocument(currentProviders);
      }
    };
  }, [documentId]);

  return { providers, synced };
};
