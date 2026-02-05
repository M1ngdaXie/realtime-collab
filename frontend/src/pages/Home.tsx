import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import type { DocumentType } from '@/api/document';
import { documentsApi } from '@/api/document';
import Navbar from '@/components/Navbar';
import { DocumentCard } from '@/components/Home/DocumentCard';
import { CreateButton } from '@/components/Home/CreateButton';
import FloatingGem from '@/components/PixelSprites/FloatingGem';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

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
      <div className="min-h-screen flex items-center justify-center bg-pixel-bg-light">
        <div className="font-pixel text-pixel-purple-bright animate-pixel-bounce">
          Loading...
        </div>
      </div>
    );
  }

  return (
    <>
      <Navbar />
      <div className="max-w-7xl mx-auto px-8 py-12 relative min-h-screen bg-pixel-bg-light">
        {/* Decorative floating gems */}
        <FloatingGem position={{ top: '20px', right: '40px' }} delay={0} size={40} />
        <FloatingGem position={{ top: '60px', left: '60px' }} delay={1.5} size={32} />
        <FloatingGem position={{ bottom: '100px', right: '100px' }} delay={3} size={36} />

        {/* Page Header */}
        <div className="mb-12">
          <h1 className="font-pixel text-4xl text-pixel-purple-bright mb-6 tracking-wide">
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
            bg-pixel-panel
            border-[3px]
            border-pixel-outline
            shadow-pixel-sm
            p-1
            mb-8
          ">
            <TabsTrigger
              value="owned"
              className="
                font-sans
                font-semibold
                data-[state=active]:bg-pixel-cyan-bright
                data-[state=active]:text-white
                data-[state=active]:shadow-pixel-sm
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
                data-[state=active]:bg-pixel-cyan-bright
                data-[state=active]:text-white
                data-[state=active]:shadow-pixel-sm
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
                <p className="col-span-full text-center text-pixel-text-muted font-sans py-12">
                  No documents yet. Create one to get started!
                </p>
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
                <p className="col-span-full text-center text-pixel-text-muted font-sans py-12">
                  No shared documents yet.
                </p>
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
