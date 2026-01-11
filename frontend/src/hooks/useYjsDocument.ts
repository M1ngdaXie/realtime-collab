import { useEffect, useState } from "react";
import { useAuth } from "../contexts/AuthContext";
import {
  createYjsDocument,
  destroyYjsDocument,
  getColorFromUserId,
  type YjsProviders,
} from "../lib/yjs";
import { useAutoSave } from "./useAutoSave";
import { documentsApi } from "../api/document";

export const useYjsDocument = (documentId: string, shareToken?: string) => {
  const { user, token } = useAuth();
  const [providers, setProviders] = useState<YjsProviders | null>(null);
  const [synced, setSynced] = useState(false);
  const [permission, setPermission] = useState<string | null>(null);
  const [role, setRole] = useState<string | null>(null);

  // Enable auto-save when providers are ready
  useAutoSave(documentId, providers?.ydoc || null);

  // Fetch permission when component mounts
  useEffect(() => {
    if (!documentId) return;

    const fetchPermission = async () => {
      try {
        const permData = await documentsApi.getPermission(documentId, shareToken);
        setPermission(permData.permission);
        setRole(permData.role);
        console.log(`📋 Permission loaded: ${permData.role} (${permData.permission})`);
      } catch (error) {
        console.error("Failed to fetch permission:", error);
      }
    };

    fetchPermission();
  }, [documentId, shareToken]);

  useEffect(() => {
    // Wait for auth (unless we have a share token for public access)
    if (!shareToken && (!user || !token)) {
      return;
    }

    let mounted = true;
    let currentProviders: YjsProviders | null = null;

    const initializeDocument = async () => {
      // For share token access, use placeholder user info
      // Extract user data (handle both direct user object and nested structure for backwards compat)
      const realUser = user || {
        id: "anonymous",
        name: "Anonymous User",
        email: "",
        avatar_url: undefined,
      };

      const currentId = realUser.id;
      const currentName = realUser.name || realUser.email || "Anonymous";
      const currentAvatar = realUser.avatar_url;

      console.log("🔍 [Debug] User data for awareness:", {
        id: currentId,
        name: currentName,
        email: realUser.email,
        avatar: currentAvatar,
        rawUser: realUser,
      });
      const authToken = token || "";
      console.log("authToken is " + token);

      const yjsProviders = await createYjsDocument(
        documentId,
        { id: currentId, name: currentName, avatar_url: currentAvatar },
        authToken,
        shareToken
      );
      currentProviders = yjsProviders;

      if (!mounted) {
        destroyYjsDocument(yjsProviders);
        return;
      }

      console.log(
        "🔍 [Debug] Full user object:",
        JSON.stringify(realUser, null, 2)
      );
      // Set user info for awareness with authenticated user data
      yjsProviders.awareness.setLocalStateField("user", {
        id: currentId,
        name: currentName,
        color: getColorFromUserId(currentId),
        avatar: currentAvatar,
      });
      console.log(currentAvatar)

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
      console.log(`[Awareness] Local user initialized: ${currentName}`, {
        color: getColorFromUserId(currentId),
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
  }, [documentId, user, token, shareToken]);


  return { providers, synced, permission, role };
};
