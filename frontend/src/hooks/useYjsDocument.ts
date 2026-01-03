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

    const initializeDocument = async () => {
      const yjsProviders = await createYjsDocument(documentId);
      currentProviders = yjsProviders;

      if (!mounted) {
        destroyYjsDocument(yjsProviders);
        return;
      }

      // Set user info for awareness
      const userName = getRandomName();
      const userColor = getRandomColor();
      yjsProviders.awareness.setLocalStateField("user", {
        name: userName,
        color: userColor,
      });

      // NEW: Add awareness event logging
      const handleAwarenessChange = ({
        added,
        updated,
        removed,
      }: {
        added: number[];
        updated: number[];
        removed: number[];
      }) => {
        const states = yjsProviders.awareness.getStates();

        added.forEach((clientId) => {
          const state = states.get(clientId);
          const user = state?.user;
          console.log(
            `[Awareness] User connected: ${
              user?.name || "Unknown"
            } (ID: ${clientId})`,
            {
              color: user?.color,
              clientId,
            }
          );
        });

        updated.forEach((clientId) => {
          const state = states.get(clientId);
          const user = state?.user;
          console.log(
            `[Awareness] User updated: ${
              user?.name || "Unknown"
            } (ID: ${clientId})`
          );
        });

        removed.forEach((clientId) => {
          console.log(`[Awareness] User disconnected (ID: ${clientId})`);
        });

        console.log(`[Awareness] Total connected users: ${states.size}`);
      };

      yjsProviders.awareness.on("change", handleAwarenessChange);

      // Listen for sync status
      yjsProviders.indexeddbProvider.on("synced", () => {
        console.log("IndexedDB synced");
        setSynced(true);
      });

      yjsProviders.websocketProvider.on(
        "status",
        (event: { status: string }) => {
          console.log("WebSocket status:", event.status);
        }
      );

      // Log local user info
      console.log(`[Awareness] Local user initialized: ${userName}`, {
        color: userColor,
        clientId: yjsProviders.awareness.clientID,
      });

      setProviders(yjsProviders);
    };

    initializeDocument();

    // Cleanup on unmount
    return () => {
      mounted = false;
      if (currentProviders) {
        console.log("[Awareness] Cleaning up local user");
        currentProviders.awareness.setLocalState(null);
        destroyYjsDocument(currentProviders);
      }
    };
  }, [documentId]);


  return { providers, synced };
};
