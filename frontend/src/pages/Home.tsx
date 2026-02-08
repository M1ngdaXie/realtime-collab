import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import type { DocumentType } from '@/api/document';
import { documentsApi } from '@/api/document';
import Navbar from '@/components/Navbar';
import { DocumentCard } from '@/components/Home/DocumentCard';
import { CreateButton } from '@/components/Home/CreateButton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import PixelIcon from '@/components/PixelIcon/PixelIcon';

const Home = () => {
  const { user } = useAuth();
  const [documents, setDocuments] = useState<DocumentType[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const navigate = useNavigate();

  const loadDocuments = async () => {
    try {
      const { documents } = await documentsApi.list();
      setDocuments(documents || []);
    } catch (error) {
      console.error('Failed to load documents:', error);
      setDocuments([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDocuments();
  }, []);

  // Filter documents into owned and shared
  const ownedDocuments = documents.filter(doc => doc.owner_id === user?.id);
  const sharedDocuments = documents.filter(doc => doc.owner_id !== user?.id);

  const createDocument = async (type: 'editor' | 'kanban') => {
    setCreating(true);
    try {
      const doc = await documentsApi.create({
        name: `New ${type === 'editor' ? 'Document' : 'Kanban Board'}`,
        type,
      });
      navigate(`/${type}/${doc.id}`);
    } catch (error) {
      console.error('Failed to create document:', error);
    } finally {
      setCreating(false);
    }
  };

  const deleteDocument = async (id: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return;

    try {
      await documentsApi.delete(id);
      loadDocuments();
    } catch (error) {
      console.error('Failed to delete document:', error);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="text-brand text-base font-medium">
          Loading...
        </div>
      </div>
    );
  }

  return (
    <>
      <Navbar />
      <div className="max-w-7xl mx-auto px-8 py-12 relative min-h-screen bg-background">

        {/* Page Header */}
        <div className="mb-12">
          <h1 className="font-display text-3xl text-text-primary mb-4">
            My Workspace
          </h1>

          {/* Create Buttons */}
          <div className="flex gap-4 mb-8 flex-wrap">
            <CreateButton
              onClick={() => createDocument('editor')}
              disabled={creating}
              icon="plus"
            >
              New Text Document
            </CreateButton>
            <CreateButton
              onClick={() => createDocument('kanban')}
              disabled={creating}
              icon="plus"
            >
              New Kanban Board
            </CreateButton>
          </div>
        </div>

        {/* Tabbed Interface */}
        <Tabs defaultValue="owned" className="w-full">
          <TabsList className="
            bg-surface-muted
            border
            border-border
            shadow-soft
            p-1
            mb-8
          ">
            <TabsTrigger
              value="owned"
              className="
                font-sans
                font-semibold
                data-[state=active]:bg-surface
                data-[state=active]:text-text-primary
                data-[state=active]:shadow-soft
                transition-all
                duration-100
              "
            >
              My Documents ({ownedDocuments.length})
            </TabsTrigger>
            <TabsTrigger
              value="shared"
              className="
                font-sans
                font-semibold
                data-[state=active]:bg-surface
                data-[state=active]:text-text-primary
                data-[state=active]:shadow-soft
                transition-all
                duration-100
              "
            >
              Shared with Me ({sharedDocuments.length})
            </TabsTrigger>
          </TabsList>

          {/* Owned Documents Tab */}
          <TabsContent value="owned">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {ownedDocuments.length === 0 ? (
                <div className="col-span-full flex flex-col items-center gap-3 text-text-muted font-sans py-12">
                  <PixelIcon name="gem" size={20} color="hsl(var(--brand-teal))" />
                  <p>No documents yet. Create one to get started!</p>
                </div>
              ) : (
                ownedDocuments.map((doc) => (
                  <DocumentCard
                    key={doc.id}
                    doc={doc}
                    onDelete={deleteDocument}
                    isShared={false}
                  />
                ))
              )}
            </div>
          </TabsContent>

          {/* Shared Documents Tab */}
          <TabsContent value="shared">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {sharedDocuments.length === 0 ? (
                <div className="col-span-full flex flex-col items-center gap-3 text-text-muted font-sans py-12">
                  <PixelIcon name="gem" size={20} color="hsl(var(--brand-teal))" />
                  <p>No shared documents yet.</p>
                </div>
              ) : (
                sharedDocuments.map((doc) => (
                  <DocumentCard
                    key={doc.id}
                    doc={doc}
                    onDelete={deleteDocument}
                    isShared={true}
                  />
                ))
              )}
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
};

export default Home;
