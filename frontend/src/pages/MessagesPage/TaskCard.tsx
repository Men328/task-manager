import { Badge, Box, Button, Group, Text } from '@mantine/core';
import { IconCircleCheck, IconCircleDashed, IconCircleX, IconProgress } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { tokens } from '../../theme';
import type { ChatTaskCategory, ChatTaskRef } from '../../types';
import classes from './MessagesPage.module.css';

const CATEGORY_COLOR: Record<string, string> = {
  todo: 'gray',
  in_progress: 'blue',
  done: 'green',
  cancelled: 'red',
};

function CategoryIcon({ category }: { category: ChatTaskCategory }) {
  const props = { size: 18, stroke: 1.8 };
  switch (category) {
    case 'done':
      return <IconCircleCheck {...props} color="var(--mantine-color-green-6)" />;
    case 'cancelled':
      return <IconCircleX {...props} color="var(--mantine-color-red-6)" />;
    case 'in_progress':
      return <IconProgress {...props} color="var(--mantine-color-blue-6)" />;
    default:
      return <IconCircleDashed {...props} color="var(--mantine-color-gray-6)" />;
  }
}

/** Thẻ task trong tin nhắn; chỉ người nhận (chủ task) mở được task. */
export function TaskCard({ task, mine, peerName }: { task: ChatTaskRef; mine: boolean; peerName: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Box className={classes.taskCard} data-category={task.statusCategory || 'todo'}>
      <Text fz={11} fw={700} tt="uppercase" c={tokens.textFaint} mb={4}>
        {mine ? t('chatTask.assignedTo', { name: peerName }) : t('chatTask.assignedBy', { name: peerName })}
      </Text>
      <Group gap={8} wrap="nowrap" align="flex-start">
        <Box pt={1}>
          <CategoryIcon category={task.statusCategory} />
        </Box>
        <Text
          fz={14}
          fw={600}
          c={tokens.text}
          className={classes.bubbleText}
          td={task.statusCategory === 'done' || task.statusCategory === 'cancelled' ? 'line-through' : undefined}
        >
          {task.title}
        </Text>
      </Group>
      <Group justify="space-between" mt={8} gap="xs">
        <Badge variant="light" color={CATEGORY_COLOR[task.statusCategory] ?? 'gray'} radius="sm">
          {task.statusName || t('chatTask.unknownStatus')}
        </Badge>
        {!mine ? (
          <Button size="compact-xs" variant="subtle" onClick={() => navigate(`/?focus=${encodeURIComponent(task.taskId)}`)}>
            {t('chatTask.open')}
          </Button>
        ) : null}
      </Group>
    </Box>
  );
}
