import { useEffect, useState } from "react";
import type { YjsProviders } from "../../lib/yjs";
import Column from "./Column.tsx";

interface KanbanBoardProps {
  providers: YjsProviders;
}

export interface Task {
  id: string;
  title: string;
  description: string;
}

export interface KanbanColumn {
  id: string;
  title: string;
  tasks: Task[];
}

const KanbanBoard = ({ providers }: KanbanBoardProps) => {
  const [columns, setColumns] = useState<KanbanColumn[]>([]);

  useEffect(() => {
    const yarray = providers.ydoc.getArray<any>("kanban-columns");

    // Initialize with default columns if empty
    if (yarray.length === 0) {
      providers.ydoc.transact(() => {
        yarray.push([
          { id: "todo", title: "To Do", tasks: [] },
          { id: "in-progress", title: "In Progress", tasks: [] },
          { id: "done", title: "Done", tasks: [] },
        ]);
      });
    }

    // Update state when Yjs array changes
    const updateColumns = () => {
      setColumns(yarray.toArray());
    };

    updateColumns();
    yarray.observe(updateColumns);

    return () => {
      yarray.unobserve(updateColumns);
    };
  }, [providers.ydoc]);

  const addTask = (columnId: string, task: Task) => {
    const yarray = providers.ydoc.getArray("kanban-columns");
    const cols = yarray.toArray();
    const columnIndex = cols.findIndex((col: any) => col.id === columnId);

    if (columnIndex !== -1) {
      providers.ydoc.transact(() => {
        const column = cols[columnIndex];
        column.tasks.push(task);
        yarray.delete(columnIndex, 1);
        yarray.insert(columnIndex, [column]);
      });
    }
  };

  const moveTask = (
    fromColumnId: string,
    toColumnId: string,
    taskId: string
  ) => {
    const yarray = providers.ydoc.getArray("kanban-columns");
    const cols = yarray.toArray();

    const fromIndex = cols.findIndex((col: any) => col.id === fromColumnId);
    const toIndex = cols.findIndex((col: any) => col.id === toColumnId);

    if (fromIndex !== -1 && toIndex !== -1) {
      providers.ydoc.transact(() => {
        const fromCol = { ...cols[fromIndex] };
        const toCol = { ...cols[toIndex] };

        const taskIndex = fromCol.tasks.findIndex((t: Task) => t.id === taskId);
        if (taskIndex !== -1) {
          const [task] = fromCol.tasks.splice(taskIndex, 1);
          toCol.tasks.push(task);

          yarray.delete(fromIndex, 1);
          yarray.insert(fromIndex, [fromCol]);
          yarray.delete(toIndex, 1);
          yarray.insert(toIndex, [toCol]);
        }
      });
    }
  };

  return (
    <div className="kanban-board">
      {columns.map((column) => (
        <Column
          key={column.id}
          column={column}
          onAddTask={(task) => addTask(column.id, task)}
          onMoveTask={(taskId, toColumnId) =>
            moveTask(column.id, toColumnId, taskId)
          }
        />
      ))}
    </div>
  );
};

export default KanbanBoard;
