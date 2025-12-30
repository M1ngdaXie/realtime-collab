import type { Task } from "./KanbanBoard.tsx";

interface CardProps {
  task: Task;
}

const Card = ({ task }: CardProps) => {
  return (
    <div className="kanban-card">
      <h4>{task.title}</h4>
      {task.description && <p>{task.description}</p>}
    </div>
  );
};

export default Card;
