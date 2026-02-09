import { useEffect, useState } from "react";
import {
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import { arrayMove } from '@dnd-kit/sortable';
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

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // Prevent accidental drags
      },
    })
  );

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
        const column = cols[columnIndex] as KanbanColumn;
        const nextTasks = [...column.tasks, task];
        const nextColumn = { ...column, tasks: nextTasks };
        yarray.delete(columnIndex, 1);
        yarray.insert(columnIndex, [nextColumn]);
      });
    }
  };

  const replaceColumn = (index: number, column: KanbanColumn) => {
    const yarray = providers.ydoc.getArray("kanban-columns");
    yarray.delete(index, 1);
    yarray.insert(index, [column]);
  };

  const findColumnByTaskId = (taskId: string) =>
    columns.find((col) => col.tasks.some((task) => task.id === taskId));

  const reorderTask = (columnId: string, fromIndex: number, toIndex: number) => {
    if (fromIndex === toIndex || fromIndex < 0 || toIndex < 0) return;
    const yarray = providers.ydoc.getArray("kanban-columns");
    const cols = yarray.toArray();
    const columnIndex = cols.findIndex((col: any) => col.id === columnId);
    if (columnIndex === -1) return;

    const column = cols[columnIndex] as KanbanColumn;
    const nextTasks = arrayMove(column.tasks, fromIndex, toIndex);
    const nextColumn = { ...column, tasks: nextTasks };

    providers.ydoc.transact(() => {
      replaceColumn(columnIndex, nextColumn);
    });
  };

  const moveTask = (
    fromColumnId: string,
    toColumnId: string,
    taskId: string,
    overTaskId?: string
  ) => {
    const yarray = providers.ydoc.getArray("kanban-columns");
    const cols = yarray.toArray();

    const fromIndex = cols.findIndex((col: any) => col.id === fromColumnId);
    const toIndex = cols.findIndex((col: any) => col.id === toColumnId);

    if (fromIndex !== -1 && toIndex !== -1) {
      providers.ydoc.transact(() => {
        const fromCol = cols[fromIndex] as KanbanColumn;
        const toCol = cols[toIndex] as KanbanColumn;
        const nextFromTasks = [...fromCol.tasks];
        const nextToTasks = fromIndex === toIndex ? nextFromTasks : [...toCol.tasks];

        const taskIndex = nextFromTasks.findIndex((t: Task) => t.id === taskId);
        if (taskIndex !== -1) {
          const [task] = nextFromTasks.splice(taskIndex, 1);
          const insertIndex =
            overTaskId && overTaskId !== toColumnId
              ? nextToTasks.findIndex((t: Task) => t.id === overTaskId)
              : -1;

          if (insertIndex >= 0) {
            nextToTasks.splice(insertIndex, 0, task);
          } else {
            nextToTasks.push(task);
          }

          const nextFromCol = { ...fromCol, tasks: nextFromTasks };
          const nextToCol = { ...toCol, tasks: nextToTasks };

          replaceColumn(fromIndex, nextFromCol);
          replaceColumn(toIndex, nextToCol);
        }
      });
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (!over) return;

    const taskId = active.id as string;
    const overId = over.id as string;

    // Find which column the task is currently in
    const fromColumn = findColumnByTaskId(taskId);
    if (!fromColumn) return;

    const overColumn =
      columns.find((col) => col.id === overId) || findColumnByTaskId(overId);
    if (!overColumn) return;

    if (fromColumn.id === overColumn.id) {
      // Reorder within the same column
      const oldIndex = fromColumn.tasks.findIndex((task) => task.id === taskId);
      const newIndex = fromColumn.tasks.findIndex((task) => task.id === overId);
      if (newIndex !== -1 && oldIndex !== -1) {
        reorderTask(fromColumn.id, oldIndex, newIndex);
      }
      return;
    }

    // Move to a different column
    moveTask(fromColumn.id, overColumn.id, taskId, overId);
  };

  return (
    <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
      <div className="kanban-board">
        {columns.map((column) => (
          <Column
            key={column.id}
            column={column}
            onAddTask={(task) => addTask(column.id, task)}
          />
        ))}
      </div>
    </DndContext>
  );
};

export default KanbanBoard;
