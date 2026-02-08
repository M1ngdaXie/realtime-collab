import { useNavigate } from 'react-router-dom';
import type { DocumentType } from '@/api/document';
import { Card } from '@/components/ui/card';
import { FileText, KanbanSquare, Trash2 } from 'lucide-react';
import PixelIcon from '@/components/PixelIcon/PixelIcon';
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
      bg-surface
      border
      border-border
      shadow-card
      hover:shadow-float
      hover:-translate-y-0.5
      transition-all
      duration-150
      p-6
      flex
      flex-col
      gap-4
    ">
      {/* Header with icon and type */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="
            bg-brand-teal
            p-2.5
            rounded-md
            shadow-soft
          ">
            <Icon className="w-5 h-5 text-white" />
          </div>
          <div>
            <h3 className="font-display text-base text-text-primary mb-1">
              {doc.name}
            </h3>
            <span className="font-sans text-xs text-text-muted">
              {typeLabel}
            </span>
          </div>
        </div>
        {isShared && (
          <span className="shared-badge">
            <PixelIcon name="gem" size={12} color="hsl(var(--brand-teal))" />
            Shared
          </span>
        )}
      </div>

      {/* Metadata */}
      <div className="font-sans text-xs text-text-muted">
        Created {new Date(doc.created_at).toLocaleDateString()}
      </div>

      {/* Actions */}
      <div className="flex gap-2 mt-auto">
        <Button
          onClick={handleOpen}
          className="
            flex-1
            bg-brand
            hover:bg-brand-dark
            text-white
            border
            border-border
            shadow-soft
            hover:shadow-card
            hover:-translate-y-0.5
            transition-all
            duration-150
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
              border
              border-border
              shadow-soft
              hover:shadow-card
              hover:-translate-y-0.5
              hover:bg-red-50
              transition-all
              duration-150
            "
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        )}
      </div>
    </Card>
  );
}
