import { useState } from 'react';
import { Box, Group } from '@mantine/core';

import { useBoard } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import KanbanColumn from './KanbanColumn';
import classes from './KanbanBoard.module.css';

interface KanbanBoardProps {
  onAddTask: (statusId: string) => void;
  onMoveTask: (taskId: string, statusId: string) => void;
}

export function KanbanBoard({ onAddTask, onMoveTask }: KanbanBoardProps) {
  const { statuses, cardsByStatus } = useBoard();
  const [draggingTaskId, setDraggingTaskId] = useState<string | null>(null);

  return (
    <Box className={`${scrollClasses.scroll} ${classes.root}`}>
      <Group align="flex-start" gap={14} wrap="nowrap" h="100%" pb={6} pr={4}>
        {statuses.map((status, index) => (
          <KanbanColumn
            key={status.id}
            status={status}
            index={index}
            tasks={cardsByStatus.get(status.id) ?? []}
            draggingTaskId={draggingTaskId}
            onDragStart={setDraggingTaskId}
            onDragEnd={() => setDraggingTaskId(null)}
            onDrop={(taskId, statusId) => {
              setDraggingTaskId(null);
              onMoveTask(taskId, statusId);
            }}
            onAddTask={onAddTask}
          />
        ))}
      </Group>
    </Box>
  );
}

export default KanbanBoard;
