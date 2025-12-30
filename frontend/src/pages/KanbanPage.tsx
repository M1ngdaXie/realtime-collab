import { useNavigate, useParams } from "react-router-dom";
import KanbanBoard from "../components/Kanban/KanbanBoard.tsx";
import UserList from "../components/Presence/UserList.tsx";
import { useYjsDocument } from "../hooks/useYjsDocument.ts";

const KanbanPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { providers, synced } = useYjsDocument(id!);

  if (!providers) {
    return <div className="loading">Connecting...</div>;
  }

  return (
    <div className="kanban-page">
      <div className="page-header">
        <button onClick={() => navigate("/")}>← Back to Home</button>
        <div className="sync-status">
          {synced ? "✓ Synced" : "⟳ Syncing..."}
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
    </div>
  );
};

export default KanbanPage;
