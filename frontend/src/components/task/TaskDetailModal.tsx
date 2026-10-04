import { Badge, Divider, Group, Modal, Stack, Text } from '@mantine/core';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useBoard } from '../../context';
import { formatDateTime, formatShortDate, hasText } from '../../lib/format';
import { statusColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import PriorityBadge from '../board/PriorityBadge';

interface TaskDetailModalProps {
  opened: boolean;
  onClose: () => void;
  task: Task | null;
}

function Label({ children }: { children: ReactNode }) {
  return (
    <Text fz={11} fw={700} c={tokens.textMuted} tt="uppercase">
      {children}
    </Text>
  );
}

export function TaskDetailModal({ opened, onClose, task }: TaskDetailModalProps) {
  const { t, i18n } = useTranslation();
  const { statusById, statuses } = useBoard();

  const status = task ? statusById.get(task.statusId) : undefined;
  const statusIndex = status ? statuses.findIndex((item) => item.id === status.id) : -1;

  return (
    <Modal opened={opened} onClose={onClose} title={t('taskDetail.title')} radius="md" size="lg">
      {task ? (
        <Stack gap="md">
          <div>
            <Text fz={18} fw={800} c={tokens.text} lh={1.3}>
              {task.title}
            </Text>
            <Group gap="xs" mt={8}>
              {status ? (
                <Badge
                  variant="light"
                  color={statusColor(status.color, statusIndex < 0 ? 0 : statusIndex)}
                >
                  {status.name}
                </Badge>
              ) : null}
              <PriorityBadge priority={task.priority} />
            </Group>
          </div>

          <Divider />

          <div>
            <Label>{t('taskDetail.description')}</Label>
            <Text fz={13} c={tokens.text} mt={4} style={{ whiteSpace: 'pre-wrap' }}>
              {hasText(task.description) ? task.description : t('taskDetail.noDescription')}
            </Text>
          </div>

          <Group gap={40}>
            <div>
              <Label>{t('taskDetail.due')}</Label>
              <Text fz={13} c={tokens.text} mt={4}>
                {formatShortDate(task.dueAt, i18n.language) ?? '—'}
              </Text>
            </div>
            <div>
              <Label>{t('taskDetail.created')}</Label>
              <Text fz={13} c={tokens.text} mt={4}>
                {formatDateTime(task.createdAt, i18n.language) ?? '—'}
              </Text>
            </div>
          </Group>

          {task.subtasks && task.subtasks.length > 0 ? (
            <div>
              <Label>{t('taskDetail.subtasks')}</Label>
              <Stack gap={4} mt={4}>
                {task.subtasks.map((sub) => (
                  <Text key={sub.id} fz={13} c={tokens.text}>
                    • {sub.title}
                  </Text>
                ))}
              </Stack>
            </div>
          ) : null}
        </Stack>
      ) : null}
    </Modal>
  );
}

export default TaskDetailModal;
