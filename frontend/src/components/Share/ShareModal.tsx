import { useState, useEffect } from 'react';
import { shareApi } from '../../api/share';
import type { DocumentShareWithUser, ShareLink } from '../../types/share';
import './ShareModal.css';

interface ShareModalProps {
  documentId: string;
  onClose: () => void;
}

function ShareModal({ documentId, onClose }: ShareModalProps) {
  const [activeTab, setActiveTab] = useState<'users' | 'link'>('users');
  const [shares, setShares] = useState<DocumentShareWithUser[]>([]);
  const [shareLink, setShareLink] = useState<ShareLink | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Form state for user sharing
  const [userEmail, setUserEmail] = useState('');
  const [permission, setPermission] = useState<'view' | 'edit'>('view');

  // Form state for link sharing
  const [linkPermission, setLinkPermission] = useState<'view' | 'edit'>('view');
  const [copied, setCopied] = useState(false);

  // Load shares on mount
  useEffect(() => {
    loadShares();
    loadShareLink();
  }, [documentId]);

  const loadShares = async () => {
    try {
      const data = await shareApi.listShares(documentId);
      setShares(data);
    } catch (err) {
      console.error('Failed to load shares:', err);
    }
  };

  const loadShareLink = async () => {
    try {
      const link = await shareApi.getShareLink(documentId);
      setShareLink(link);
      if (link) {
        setLinkPermission(link.permission);
      }
    } catch (err) {
      console.error('Failed to load share link:', err);
    }
  };

  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!userEmail.trim()) {
      setError('Please enter an email address');
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      await shareApi.createShare(documentId, {
        user_email: userEmail,
        permission,
      });

      setSuccess('User added successfully');
      setUserEmail('');
      await loadShares();
    } catch (err) {
      setError('Failed to add user. Make sure the email is registered.');
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveUser = async (userId: string) => {
    if (!confirm('Remove access for this user?')) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await shareApi.deleteShare(documentId, userId);
      setSuccess('User removed successfully');
      await loadShares();
    } catch (err) {
      setError('Failed to remove user');
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateLink = async () => {
    setLoading(true);
    setError(null);

    try {
      const link = await shareApi.createShareLink(documentId, linkPermission);
      setShareLink(link);
      setSuccess('Share link created');
    } catch (err) {
      setError('Failed to create share link');
    } finally {
      setLoading(false);
    }
  };

  const handleRevokeLink = async () => {
    if (!confirm('Revoke this share link? Anyone with the link will lose access.')) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await shareApi.revokeShareLink(documentId);
      setShareLink(null);
      setSuccess('Share link revoked');
    } catch (err) {
      setError('Failed to revoke share link');
    } finally {
      setLoading(false);
    }
  };

  const handleCopyLink = () => {
    if (!shareLink) return;

    const url = `${window.location.origin}/editor/${documentId}?share=${shareLink.token}`;
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Share Document</h2>
          <button className="close-button" onClick={onClose}>×</button>
        </div>

        <div className="tabs">
          <button
            className={`tab ${activeTab === 'users' ? 'active' : ''}`}
            onClick={() => setActiveTab('users')}
          >
            Share with Users
          </button>
          <button
            className={`tab ${activeTab === 'link' ? 'active' : ''}`}
            onClick={() => setActiveTab('link')}
          >
            Public Link
          </button>
        </div>

        {error && <div className="message error">{error}</div>}
        {success && <div className="message success">{success}</div>}

        {activeTab === 'users' && (
          <div className="tab-content">
            <form onSubmit={handleAddUser} className="share-form">
              <input
                type="email"
                placeholder="Email address"
                value={userEmail}
                onChange={(e) => setUserEmail(e.target.value)}
                className="share-input"
                disabled={loading}
              />
              <select
                value={permission}
                onChange={(e) => setPermission(e.target.value as 'view' | 'edit')}
                className="share-select"
                disabled={loading}
              >
                <option value="view">Can view</option>
                <option value="edit">Can edit</option>
              </select>
              <button type="submit" className="share-button" disabled={loading}>
                {loading ? 'Adding...' : 'Add User'}
              </button>
            </form>

            <div className="shares-list">
              <h3>People with access</h3>
              {shares.length === 0 ? (
                <p className="empty-state">No users have been given access yet.</p>
              ) : (
                shares.map((share) => (
                  <div key={share.id} className="share-item">
                    <div className="share-user">
                      {share.user.avatar_url && (
                        <img
                          src={share.user.avatar_url}
                          alt={share.user.name}
                          className="share-avatar"
                        />
                      )}
                      <div className="share-info">
                        <div className="share-name">{share.user.name}</div>
                        <div className="share-email">{share.user.email}</div>
                      </div>
                    </div>
                    <div className="share-actions">
                      <span className="permission-badge">{share.permission}</span>
                      <button
                        onClick={() => handleRemoveUser(share.user_id)}
                        className="remove-button"
                        disabled={loading}
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {activeTab === 'link' && (
          <div className="tab-content">
            {!shareLink ? (
              <div className="link-creation">
                <p>Create a public link that anyone can use to access this document.</p>
                <div className="link-form">
                  <select
                    value={linkPermission}
                    onChange={(e) => setLinkPermission(e.target.value as 'view' | 'edit')}
                    className="share-select"
                    disabled={loading}
                  >
                    <option value="view">Can view</option>
                    <option value="edit">Can edit</option>
                  </select>
                  <button
                    onClick={handleGenerateLink}
                    className="share-button"
                    disabled={loading}
                  >
                    {loading ? 'Creating...' : 'Generate Link'}
                  </button>
                </div>
              </div>
            ) : (
              <div className="link-display">
                <p>Anyone with this link can {shareLink.permission} this document.</p>
                <div className="link-box">
                  <input
                    type="text"
                    value={`${window.location.origin}/editor/${documentId}?share=${shareLink.token}`}
                    readOnly
                    className="link-input"
                  />
                  <button onClick={handleCopyLink} className="copy-button">
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <div className="link-meta">
                  <span className="permission-badge">{shareLink.permission}</span>
                  <span className="link-date">
                    Created {new Date(shareLink.created_at).toLocaleDateString()}
                  </span>
                </div>
                <button
                  onClick={handleRevokeLink}
                  className="revoke-button"
                  disabled={loading}
                >
                  {loading ? 'Revoking...' : 'Revoke Link'}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default ShareModal;
