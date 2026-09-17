import { useState } from 'react';
import { ActionIcon, Box, Group, Menu, Stack, Text, Tooltip, UnstyledButton } from '@mantine/core';
import { IconDots, IconPlus, IconArrowsExchange } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { statusColor } from '../../lib/tokens';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { Task, TaskStatus } from '../../types';
import TaskCard from './TaskCard';
import classes from './KanbanColumn.module.css';

interface KanbanColumnProps {
  status: TaskStatus;
  index: number;
  tasks: Task[];
  draggingTaskId: string | null;
  onDragStart: (taskId: string) => void;
  onDragEnd: () => void;
  onDrop: (taskId: string, statusId: string) => void;
  onAddTask: (statusId: string) => void;
}

export function KanbanColumn({
  status,
  index,
  tasks,
  draggingTaskId,
  onDragStart,
  onDragEnd,
  onDrop,
  onAddTask,
}: KanbanColumnProps) {
  const { t } = useTranslation();
  const [isOver, setIsOver] = useState(false);
  const navigate = useNavigate();
  const color = statusColor(status.color, index);

  return (
    <Box
      className={classes.column}
      data-drop={isOver}
      w={272}
      onDragOver={(event) => {
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        if (!isOver) {
          setIsOver(true);
        }
      }}
      onDragLeave={(event) => {
        const next = event.relatedTarget as Node | null;
        if (next && event.currentTarget.contains(next)) {
          return;
        }
        setIsOver(false);
      }}
      onDrop={(event) => {
        event.preventDefault();
        setIsOver(false);
        const taskId = event.dataTransfer.getData('text/plain');
        if (taskId) {
          onDrop(taskId, status.id);
        }
      }}
    >
      <Group justify="space-between" wrap="nowrap" mb={10} px={2}>
        <Group gap={7} wrap="nowrap">
          <Box className={classes.dot} style={{ background: color }} />
          <Text fz={13.5} fw={700} c={tokens.text}>
            {status.name}
          </Text>
          <Box px={6} py={1} className={classes.count}>
            <Text fz={10.5} fw={700} c={tokens.textMuted}>
              {tasks.length}
            </Text>
          </Box>
        </Group>

        <Group gap={2} wrap="nowrap">
          <Menu position="bottom-end" withArrow shadow="md">
            <Menu.Target>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                aria-label={t('kanban.columnOptions', { name: status.name })}
              >
                <IconDots size={16} style={{ color: tokens.textMuted }} />
              </ActionIcon>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item leftSection={<IconPlus size={14} />} onClick={() => onAddTask(status.id)}>
                {t('kanban.addToColumn')}
              </Menu.Item>
              <Menu.Item
                leftSection={<IconArrowsExchange size={14} />}
                onClick={() => navigate('/statuses')}
              >
                {t('kanban.viewLifecycle')}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>

          <Tooltip label={t('kanban.addTask')} withArrow openDelay={300}>
            <ActionIcon
              variant="subtle"
              color="gray"
              size="sm"
              onClick={() => onAddTask(status.id)}
              aria-label={t('kanban.addTaskTo', { name: status.name })}
            >
              <IconPlus size={15} style={{ color: tokens.textMuted }} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Group>

      <Box className={`${scrollClasses.scroll} ${classes.list}`}>
        <Stack gap={10}>
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              dragging={draggingTaskId === task.id}
              onDragStart={onDragStart}
              onDragEnd={onDragEnd}
            />
          ))}

          {tasks.length === 0 ? (
            <Box className={classes.empty} data-over={isOver}>
              <Text fz={11.5} c={tokens.textFaint}>
                {isOver ? t('kanban.dropHere') : t('kanban.empty')}
              </Text>
            </Box>
          ) : null}
        </Stack>
      </Box>

      <UnstyledButton onClick={() => onAddTask(status.id)} mt={10} py={8} className={classes.addButton}>
        <Group gap={5} justify="center" wrap="nowrap">
          <IconPlus size={14} style={{ color: tokens.textMuted }} />
          <Text fz={12} fw={600} c={tokens.textMuted}>
            {t('kanban.addTaskCta')}
          </Text>
        </Group>
      </UnstyledButton>
    </Box>
  );
}

export default KanbanColumn;
