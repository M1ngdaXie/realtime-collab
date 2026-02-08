import { Eye } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import Editor from "../components/Editor/Editor.tsx";
import Navbar from "../components/Navbar.tsx";
import UserList from "../components/Presence/UserList.tsx";
import ShareModal from "../components/Share/ShareModal.tsx";
import VersionHistoryPanel from "../components/VersionHistory/VersionHistoryPanel.tsx";
import { useYjsDocument } from "../hooks/useYjsDocument.ts";

const EditorPage = () => {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const shareToken = searchParams.get('share') || undefined;
  const { providers, permission, role } = useYjsDocument(id!, shareToken);
  const [showShareModal, setShowShareModal] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);

  if (!providers) {
    return <div className="loading">Connecting...</div>;
  }

  return (
    <div className="editor-page">
      <Navbar />
      <div className="page-header">
        <button onClick={() => navigate("/")}>← Back to Home</button>
        <div className="header-actions">
          {permission !== "edit" && permission !== null && (
            <div className="view-only-badge">
              <Eye className="view-only-icon" />
              <span>View only</span>
            </div>
          )}
          {!shareToken && (
            <button className="share-btn" onClick={() => setShowShareModal(true)}>
              Share
            </button>
          )}
          {!shareToken && (
            <button className="history-btn" onClick={() => setShowVersionHistory(true)}>
              History
            </button>
          )}
        </div>
      </div>

      <div className="page-content">
        <div className="main-area">
          <Editor providers={providers} permission={permission} />
        </div>
        <div className="sidebar">
          <UserList awareness={providers.awareness} />
        </div>
      </div>

      {showShareModal && (
        <ShareModal
          documentId={id!}
          onClose={() => setShowShareModal(false)}
          currentPermission={permission || undefined}
          currentRole={role || undefined}
        />
      )}

      {showVersionHistory && (
        <VersionHistoryPanel
          documentId={id!}
          ydoc={providers.ydoc}
          canEdit={permission === "edit"}
          onClose={() => setShowVersionHistory(false)}
        />
      )}
    </div>
  );
};

export default EditorPage;
