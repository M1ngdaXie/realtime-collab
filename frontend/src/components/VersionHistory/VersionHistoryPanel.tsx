import { useState, useEffect } from 'react';
import { versionsApi, type DocumentVersion } from '../../api/document';
import * as Y from 'yjs';
import './VersionHistoryPanel.css';

interface VersionHistoryPanelProps {
  documentId: string;
  ydoc: Y.Doc | null;
  canEdit: boolean;
  onClose: () => void;
}

function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} hours ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)} days ago`;

  return date.toLocaleDateString();
}

function VersionHistoryPanel({ documentId, ydoc, canEdit, onClose }: VersionHistoryPanelProps) {
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [restoring, setRestoring] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [versionLabel, setVersionLabel] = useState('');
  const [creating, setCreating] = useState(false);

  const LIMIT = 20;

  useEffect(() => {
    loadVersions();
  }, [documentId, offset]);

  const loadVersions = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await versionsApi.list(documentId, LIMIT, offset);
      setVersions(data.versions || []);
      setTotal(data.total);
    } catch (err) {
      console.error('Failed to load versions:', err);
      setError('Failed to load version history');
    } finally {
      setLoading(false);
    }
  };

  const handleRestore = async (version: DocumentVersion) => {
    if (!canEdit) {
      alert('You need edit permission to restore versions');
      return;
    }

    if (!confirm(`Restore to version ${version.version_number}? This will create a new version from the restored content.`)) {
      return;
    }

    setRestoring(version.id);
    try {
      await versionsApi.restore(documentId, version.id);

      // Clear IndexedDB cache so restored state is used on reload
      // (y-indexeddb uses documentId as the database name)
      await new Promise<void>((resolve, reject) => {
        const request = indexedDB.deleteDatabase(documentId);
        request.onsuccess = () => resolve();
        request.onerror = () => reject(request.error);
      });

      alert('Version restored successfully! The page will reload to sync changes.');
      window.location.reload();
    } catch (err) {
      console.error('Failed to restore version:', err);
      alert('Failed to restore version');
    } finally {
      setRestoring(null);
    }
  };

  const handleCreateVersion = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!ydoc) {
      alert('Document not loaded');
      return;
    }

    setCreating(true);
    try {
      // Get Yjs state
      const yjsSnapshot = Y.encodeStateAsUpdate(ydoc);

      // Extract text preview from document (use placeholder if empty)
      const textPreview = extractTextFromYjs(ydoc) || '(empty document)';

      await versionsApi.create(documentId, yjsSnapshot, textPreview, versionLabel || undefined);

      setShowCreateModal(false);
      setVersionLabel('');
      await loadVersions();
      alert('Version created successfully!');
    } catch (err) {
      console.error('Failed to create version:', err);
      alert('Failed to create version');
    } finally {
      setCreating(false);
    }
  };

  const extractTextFromYjs = (doc: Y.Doc): string => {
    try {
      const xmlFragment = doc.getXmlFragment('default');
      let text = '';

      const extractText = (item: Y.XmlElement | Y.XmlText | Y.Item): void => {
        if (item instanceof Y.XmlText) {
          text += item.toString();
        } else if (item instanceof Y.XmlElement) {
          // Add newline for block elements
          if (['paragraph', 'heading', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6'].includes(item.nodeName)) {
            if (text.length > 0) text += '\n';
          }
          item.toArray().forEach((child) => extractText(child as Y.XmlElement | Y.XmlText));
        }
      };

      xmlFragment.toArray().forEach((item) => extractText(item as Y.XmlElement | Y.XmlText));
      return text.trim();
    } catch (err) {
      console.error('Failed to extract text:', err);
      return '';
    }
  };

  const handleNextPage = () => {
    if (offset + LIMIT < total) {
      setOffset(offset + LIMIT);
    }
  };

  const handlePrevPage = () => {
    if (offset > 0) {
      setOffset(Math.max(0, offset - LIMIT));
    }
  };

  return (
    <div className="version-panel-overlay" onClick={onClose}>
      <div className="version-panel" onClick={(e) => e.stopPropagation()}>
        <div className="version-panel-header">
          <h2>Version History</h2>
          <button className="close-button" onClick={onClose}>&times;</button>
        </div>

        {canEdit && (
          <div className="version-panel-actions">
            <button
              className="create-version-btn"
              onClick={() => setShowCreateModal(true)}
            >
              + Save Current Version
            </button>
          </div>
        )}

        <div className="version-panel-content">
          {error && <div className="message error">{error}</div>}

          {loading ? (
            <div className="loading-state">Loading versions...</div>
          ) : versions.length === 0 ? (
            <div className="empty-state">
              <p>No versions yet</p>
              <small>Versions are created when you save manually or automatically over time</small>
            </div>
          ) : (
            <div className="version-list">
              {versions.map((version) => (
                <div key={version.id} className="version-item">
                  <div className="version-header">
                    <span className="version-number">v{version.version_number}</span>
                    {version.is_auto_generated && (
                      <span className="auto-badge">Auto</span>
                    )}
                    {version.version_label && (
                      <span className="version-label">{version.version_label}</span>
                    )}
                  </div>

                  <div className="version-meta">
                    {version.author ? (
                      <div className="version-author">
                        {version.author.avatar_url && (
                          <img
                            src={version.author.avatar_url}
                            alt={version.author.name}
                            className="author-avatar"
                          />
                        )}
                        <span>{version.author.name || version.author.email}</span>
                      </div>
                    ) : (
                      <span className="version-author">Unknown user</span>
                    )}
                    <span className="version-time">{formatTimeAgo(version.created_at)}</span>
                  </div>

                  {version.text_preview && (
                    <div className="version-preview">
                      {version.text_preview.length > 150
                        ? version.text_preview.substring(0, 150) + '...'
                        : version.text_preview}
                    </div>
                  )}

                  {canEdit && (
                    <button
                      className="restore-btn"
                      onClick={() => handleRestore(version)}
                      disabled={restoring === version.id}
                    >
                      {restoring === version.id ? 'Restoring...' : 'Restore'}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}

          {total > LIMIT && (
            <div className="pagination">
              <button onClick={handlePrevPage} disabled={offset === 0}>
                &larr; Previous
              </button>
              <span>
                {offset + 1}-{Math.min(offset + LIMIT, total)} of {total}
              </span>
              <button onClick={handleNextPage} disabled={offset + LIMIT >= total}>
                Next &rarr;
              </button>
            </div>
          )}
        </div>

        {/* Create Version Modal */}
        {showCreateModal && (
          <div className="create-modal-overlay" onClick={() => setShowCreateModal(false)}>
            <div className="create-modal" onClick={(e) => e.stopPropagation()}>
              <h3>Save Version</h3>
              <p>Create a snapshot of the current document state.</p>

              <form onSubmit={handleCreateVersion}>
                <label htmlFor="version-label">Version Label (optional)</label>
                <input
                  id="version-label"
                  type="text"
                  value={versionLabel}
                  onChange={(e) => setVersionLabel(e.target.value)}
                  placeholder="e.g., Before major changes"
                  maxLength={100}
                />

                <div className="modal-buttons">
                  <button
                    type="button"
                    onClick={() => setShowCreateModal(false)}
                    disabled={creating}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="primary"
                    disabled={creating}
                  >
                    {creating ? 'Creating...' : 'Save Version'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default VersionHistoryPanel;
