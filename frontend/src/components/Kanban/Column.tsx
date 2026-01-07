import { useState } from "react";
import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import Card from "./Card.tsx";
import type { KanbanColumn, Task } from "./KanbanBoard.tsx";

interface ColumnProps {
  column: KanbanColumn;
  onAddTask: (task: Task) => void;
  onMoveTask: (taskId: string, toColumnId: string) => void;
}

const Column = ({ column, onAddTask }: ColumnProps) => {
  const [isAdding, setIsAdding] = useState(false);
  const [newTaskTitle, setNewTaskTitle] = useState("");

  const { setNodeRef, isOver } = useDroppable({
    id: column.id,
  });

  const handleAddTask = () => {
    if (newTaskTitle.trim()) {
      onAddTask({
        id: `task-${Date.now()}`,
        title: newTaskTitle,
        description: "",
      });
      setNewTaskTitle("");
      setIsAdding(false);
    }
  };

  return (
    <div
      ref={setNodeRef}
      className={`kanban-column ${isOver ? 'column-drag-over' : ''}`}
    >
      <h3 className="column-title">{column.title}</h3>
      <div className="column-content">
        <SortableContext
          items={column.tasks.map(t => t.id)}
          strategy={verticalListSortingStrategy}
        >
          {column.tasks.map((task) => (
            <Card key={task.id} task={task} />
          ))}
        </SortableContext>

        {isAdding ? (
          <div className="add-task-form">
            <input
              type="text"
              value={newTaskTitle}
              onChange={(e) => setNewTaskTitle(e.target.value)}
              placeholder="Task title..."
              autoFocus
              onKeyPress={(e) => e.key === "Enter" && handleAddTask()}
            />
            <div className="form-actions">
              <button onClick={handleAddTask}>Add</button>
              <button onClick={() => setIsAdding(false)}>Cancel</button>
            </div>
          </div>
        ) : (
          <button className="add-task-btn" onClick={() => setIsAdding(true)}>
            + Add Task
          </button>
        )}
      </div>
    </div>
  );
};

export default Column;
