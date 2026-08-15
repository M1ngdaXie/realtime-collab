import { useEffect, useRef } from 'react';
import * as Y from 'yjs';
import { documentsApi } from '../api/document';

export const useAutoSave = (documentId: string, ydoc: Y.Doc | null, shareToken?: string) => {
  const saveTimeoutRef = useRef<number | null>(null);
  const isSavingRef = useRef(false);

  useEffect(() => {
    if (!ydoc) return;

    const handleUpdate = (_update: Uint8Array, origin: any) => {
      // Ignore updates from initial sync or remote sources
      if (origin === 'init' || origin === 'remote') return;

      // Clear existing timeout
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }

      // Set new timeout: save 2 seconds after last edit
      saveTimeoutRef.current = setTimeout(async () => {
        if (isSavingRef.current) return; // Prevent concurrent saves

        isSavingRef.current = true;
        try {
          const state = Y.encodeStateAsUpdate(ydoc);
          await documentsApi.updateState(documentId, state, shareToken);
          console.log('✓ Document saved to database');
        } catch (error) {
          console.error('Failed to save document:', error);
        } finally {
          isSavingRef.current = false;
        }
      }, 2000); // 2 second debounce
    };

    ydoc.on('update', handleUpdate);

    // Cleanup
    return () => {
      ydoc.off('update', handleUpdate);
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, [documentId, ydoc, shareToken]);
};
