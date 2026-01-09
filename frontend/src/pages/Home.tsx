import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { DocumentType } from "../api/document.ts";
import { documentsApi } from "../api/document.ts";
import Navbar from "../components/Navbar.tsx";
import PixelIcon from "../components/PixelIcon/PixelIcon.tsx";
import FloatingGem from "../components/PixelSprites/FloatingGem.tsx";

const Home = () => {
  const [documents, setDocuments] = useState<DocumentType[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const navigate = useNavigate();

  const loadDocuments = async () => {
    try {
      const { documents } = await documentsApi.list();
      setDocuments(documents || []);
    } catch (error) {
      console.error("Failed to load documents:", error);
      setDocuments([]); // Set empty array on error to prevent null access
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDocuments();
  }, []);

  const createDocument = async (type: "editor" | "kanban") => {
    setCreating(true);
    try {
      const doc = await documentsApi.create({
        name: `New ${type === "editor" ? "Document" : "Kanban Board"}`,
        type,
      });
      navigate(`/${type}/${doc.id}`);
    } catch (error) {
      console.error("Failed to create document:", error);
    } finally {
      setCreating(false);
    }
  };

  const deleteDocument = async (id: string) => {
    if (!confirm("Are you sure you want to delete this document?")) return;

    try {
      await documentsApi.delete(id);
      loadDocuments();
    } catch (error) {
      console.error("Failed to delete document:", error);
    }
  };

  if (loading) {
    return <div className="loading">Loading documents...</div>;
  }

  return (
    <>
      <Navbar />
      <div className="home-page" style={{ position: 'relative' }}>
        <FloatingGem position={{ top: '20px', right: '40px' }} delay={0} size={40} />
        <FloatingGem position={{ top: '60px', left: '60px' }} delay={1.5} size={32} />
        <FloatingGem position={{ bottom: '100px', right: '100px' }} delay={3} size={36} />

        <h1>My Documents</h1>

      <div className="create-buttons">
        <button onClick={() => createDocument("editor")} disabled={creating}>
          <PixelIcon name="plus" size={20} />
          <span style={{ marginLeft: '8px' }}>New Text Document</span>
        </button>
        <button onClick={() => createDocument("kanban")} disabled={creating}>
          <PixelIcon name="plus" size={20} />
          <span style={{ marginLeft: '8px' }}>New Kanban Board</span>
        </button>
      </div>

      <div className="document-list">
        {documents.length === 0 ? (
          <p>No documents yet. Create one to get started!</p>
        ) : (
          documents.map((doc) => (
            <div key={doc.id} className="document-card">
              <div className="doc-info">
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                  <PixelIcon name={doc.type === 'editor' ? 'document' : 'kanban'} size={24} color="var(--pixel-purple-bright)" />
                  <h3 style={{ margin: 0 }}>{doc.name}</h3>
                </div>
                <p className="doc-type">{doc.type}</p>
                <p className="doc-date">
                  Created: {new Date(doc.created_at).toLocaleDateString()}
                </p>
              </div>
              <div className="doc-actions">
                <button onClick={() => navigate(`/${doc.type}/${doc.id}`)} aria-label={`Open ${doc.name}`}>
                  <PixelIcon name="back-arrow" size={16} style={{ transform: 'rotate(180deg)' }} />
                  <span style={{ marginLeft: '6px' }}>Open</span>
                </button>
                <button onClick={() => deleteDocument(doc.id)} aria-label={`Delete ${doc.name}`}>
                  <PixelIcon name="trash" size={16} />
                  <span style={{ marginLeft: '6px' }}>Delete</span>
                </button>
              </div>
            </div>
          ))
        )}
      </div>
      </div>
    </>
  );
};

export default Home;
