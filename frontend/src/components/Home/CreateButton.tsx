import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Plus, FileText, KanbanSquare } from 'lucide-react';

interface CreateButtonProps {
  onClick: () => void;
  disabled?: boolean;
  icon: 'plus' | 'document' | 'kanban';
  children: ReactNode;
}

const iconMap = {
  plus: Plus,
  document: FileText,
  kanban: KanbanSquare,
};

export function CreateButton({ onClick, disabled, icon, children }: CreateButtonProps) {
  const Icon = iconMap[icon];

  return (
    <Button
      onClick={onClick}
      disabled={disabled}
      className="
        bg-pixel-purple-bright
        hover:bg-pixel-purple-deep
        text-white
        border-[3px]
        border-pixel-outline
        shadow-pixel
        hover:shadow-pixel-hover
        hover:-translate-y-0.5
        hover:-translate-x-0.5
        active:translate-y-0.5
        active:translate-x-0.5
        active:shadow-pixel-active
        transition-all
        duration-75
        font-sans
        font-semibold
        px-6
        py-3
        flex
        items-center
        gap-2
        disabled:opacity-60
        disabled:cursor-not-allowed
      "
    >
      <Icon className="w-5 h-5" />
      {children}
    </Button>
  );
}
