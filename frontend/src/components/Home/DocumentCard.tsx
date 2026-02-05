import { useNavigate } from 'react-router-dom';
import type { DocumentType } from '@/api/document';
import { Card } from '@/components/ui/card';
import { FileText, KanbanSquare, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface DocumentCardProps {
  doc: DocumentType;
  onDelete: (id: string) => void;
  isShared?: boolean;
}

export function DocumentCard({ doc, onDelete, isShared }: DocumentCardProps) {
  const navigate = useNavigate();

  const Icon = doc.type === 'editor' ? FileText : KanbanSquare;
  const typeLabel = doc.type === 'editor' ? 'Text Document' : 'Kanban Board';

  const handleOpen = () => {
    navigate(`/${doc.type}/${doc.id}`);
  };

  return (
    <Card className="
      group
      relative
      bg-pixel-white
      border-[3px]
      border-pixel-outline
      shadow-pixel-md
      hover:shadow-pixel-lg
      hover:-translate-y-0.5
      hover:-translate-x-0.5
      transition-all
      duration-100
      p-6
      flex
      flex-col
      gap-4
    ">
      {/* Header with icon and type */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="
            bg-pixel-cyan-bright
            p-2
            border-[2px]
            border-pixel-outline
            shadow-pixel-sm
          ">
            <Icon className="w-6 h-6 text-white" />
          </div>
          <div>
            <h3 className="font-pixel text-sm text-pixel-text-primary mb-1">
              {doc.name}
            </h3>
            <span className="font-sans text-xs text-pixel-text-muted">
              {typeLabel}
            </span>
          </div>
        </div>
        {isShared && (
          <span className="
            bg-pixel-pink-vibrant
            text-white
            font-sans
            text-xs
            font-semibold
            px-2
            py-1
            border-[2px]
            border-pixel-outline
          ">
            Shared
          </span>
        )}
      </div>

      {/* Metadata */}
      <div className="font-sans text-xs text-pixel-text-muted">
        Created {new Date(doc.created_at).toLocaleDateString()}
      </div>

      {/* Actions */}
      <div className="flex gap-2 mt-auto">
        <Button
          onClick={handleOpen}
          className="
            flex-1
            bg-pixel-cyan-bright
            hover:bg-pixel-purple-bright
            text-white
            border-[2px]
            border-pixel-outline
            shadow-pixel-sm
            hover:shadow-pixel-hover
            hover:-translate-y-0.5
            hover:-translate-x-0.5
            transition-all
            duration-75
            font-sans
            font-semibold
          "
        >
          Open
        </Button>
        {!isShared && (
          <Button
            onClick={() => onDelete(doc.id)}
            variant="outline"
            className="
              border-[2px]
              border-pixel-outline
              shadow-pixel-sm
              hover:shadow-pixel-hover
              hover:-translate-y-0.5
              hover:-translate-x-0.5
              hover:bg-red-50
              transition-all
              duration-75
            "
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        )}
      </div>
    </Card>
  );
}
