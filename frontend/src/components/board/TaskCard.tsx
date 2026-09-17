import { Box, Divider, Group, Text } from '@mantine/core';
import { IconAlertTriangle, IconCalendar, IconChecklist } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { useBoard } from '../../context';
import { derivedProgress, formatShortDate, hasText, isOverdue } from '../../lib/format';
import { statusColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import PriorityBadge from './PriorityBadge';
import TaskProgress from './TaskProgress';
import classes from './TaskCard.module.css';

interface TaskCardProps {
  task: Task;
  dragging: boolean;
  onDragStart: (taskId: string) => void;
  onDragEnd: () => void;
}

export function TaskCard({ task, dragging, onDragStart, onDragEnd }: TaskCardProps) {
  const { i18n } = useTranslation();
  const { statusById, taskById, isStatusDone, statuses } = useBoard();

  const status = statusById.get(task.statusId);
  const statusIndex = statuses.findIndex((item) => item.id === task.statusId);
  const parent = hasText(task.parentTaskId) ? taskById.get(task.parentTaskId as string) : undefined;

  const progress = derivedProgress(task.subtasks, isStatusDone);
  const subtasks = task.subtasks ?? [];
  const doneSubtasks = subtasks.filter((sub) => isStatusDone(sub.statusId)).length;

  const due = formatShortDate(task.dueAt, i18n.language);
  const overdue = isOverdue(task.dueAt);

  return (
    <Box
      className={classes.card}
      data-dragging={dragging}
      draggable
      onDragStart={(event) => {
        event.dataTransfer.setData('text/plain', task.id);
        event.dataTransfer.effectAllowed = 'move';
        onDragStart(task.id);
      }}
      onDragEnd={onDragEnd}
      p={13}
    >
      <Group justify="space-between" align="flex-start" gap={8} wrap="nowrap">
        <PriorityBadge priority={task.priority} />
        {due ? (
          <Group gap={4} wrap="nowrap">
            {overdue ? (
              <IconAlertTriangle size={12} style={{ color: tokens.danger }} />
            ) : (
              <IconCalendar size={12} style={{ color: tokens.textFaint }} />
            )}
            <Text fz={10.5} fw={600} c={overdue ? tokens.danger : tokens.textMuted}>
              {due}
            </Text>
          </Group>
        ) : null}
      </Group>

      <Box mt={9}>
        {parent ? (
          <Group gap={6} wrap="nowrap" align="center" mb={5}>
            <Box
              className={classes.parentDot}
              style={{ background: statusColor(status?.color, statusIndex < 0 ? 0 : statusIndex) }}
            />
            <Text fz={11} fw={700} c={tokens.accentTag} truncate>
              {parent.title}
            </Text>
          </Group>
        ) : null}

        <Text fz={14} fw={700} c={tokens.text} lh={1.35}>
          {task.title}
        </Text>

        {hasText(task.description) ? (
          <Text fz={11.5} c={tokens.textMuted} mt={5} lh={1.5} className={classes.description}>
            {task.description}
          </Text>
        ) : null}
      </Box>

      {progress !== null ? (
        <Box mt={12}>
          <TaskProgress value={progress} />
        </Box>
      ) : null}

      <Divider my={11} color={tokens.border} />

      <Group gap={10} wrap="nowrap">
        {subtasks.length > 0 ? (
          <Group gap={4} wrap="nowrap">
            <IconChecklist size={13} style={{ color: tokens.textFaint }} />
            <Text fz={11} fw={600} c={tokens.textMuted}>
              {doneSubtasks}/{subtasks.length}
            </Text>
          </Group>
        ) : (
          <Text fz={11} c={tokens.textFaint}>
            {formatShortDate(task.createdAt, i18n.language) ?? '—'}
          </Text>
        )}
      </Group>
    </Box>
  );
}

export default TaskCard;
