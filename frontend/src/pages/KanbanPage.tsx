import { useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import KanbanBoard from "../components/Kanban/KanbanBoard.tsx";
import Navbar from "../components/Navbar.tsx";
import UserList from "../components/Presence/UserList.tsx";
import ShareModal from "../components/Share/ShareModal.tsx";
import { useYjsDocument } from "../hooks/useYjsDocument.ts";

const KanbanPage = () => {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const shareToken = searchParams.get('share') || undefined;
  const { providers } = useYjsDocument(id!, shareToken);
  const [showShareModal, setShowShareModal] = useState(false);

  if (!providers) {
    return <div className="loading">Connecting...</div>;
  }

  return (
    <div className="kanban-page">
      <Navbar />
      <div className="page-header">
        <button onClick={() => navigate("/")}>← Back to Home</button>
        <div className="header-actions">
          {!shareToken && (
            <button className="share-btn" onClick={() => setShowShareModal(true)}>
              Share
            </button>
          )}
        </div>
      </div>

      <div className="page-content">
        <div className="main-area">
          <KanbanBoard providers={providers} />
        </div>
        <div className="sidebar">
          <UserList awareness={providers.awareness} />
        </div>
      </div>

      {showShareModal && (
        <ShareModal documentId={id!} onClose={() => setShowShareModal(false)} />
      )}
    </div>
  );
};

export default KanbanPage;
