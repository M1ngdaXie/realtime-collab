import { useState } from "react";
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
    <div className="kanban-column">
      <h3 className="column-title">{column.title}</h3>
      <div className="column-content">
        {column.tasks.map((task) => (
          <Card key={task.id} task={task} />
        ))}

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
