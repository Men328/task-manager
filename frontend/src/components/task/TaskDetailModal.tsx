import { Badge, Divider, Group, Modal, Stack, Text } from '@mantine/core';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useBoard } from '../../context';
import { derivedProgress, formatDateTime, formatShortDate, hasText } from '../../lib/format';
import { statusColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import AttachmentSection from '../attachment/AttachmentSection';
import PriorityBadge from '../board/PriorityBadge';
import TaskProgress from '../board/TaskProgress';

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
  const { statusById, statuses, taskById, isStatusDone } = useBoard();

  const status = task ? statusById.get(task.statusId) : undefined;
  const statusIndex = status ? statuses.findIndex((item) => item.id === status.id) : -1;
  const parent = task && hasText(task.parentTaskId)
    ? taskById.get(task.parentTaskId as string)
    : undefined;
  const progress = task ? derivedProgress(task.subtasks, isStatusDone) : null;

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

          <Group gap={40} align="flex-start">
            <div>
              <Label>{t('taskDetail.status')}</Label>
              <Text fz={13} c={tokens.text} mt={4}>
                {status?.name ?? '—'}
              </Text>
            </div>
            <div>
              <Label>{t('taskDetail.start')}</Label>
              <Text fz={13} c={tokens.text} mt={4}>
                {formatShortDate(task.startAt, i18n.language) ?? '—'}
              </Text>
            </div>
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

          {progress !== null ? (
            <div>
              <Label>{t('taskDetail.progress')}</Label>
              <Stack gap={4} mt={4}>
                <TaskProgress value={progress} />
              </Stack>
            </div>
          ) : null}

          {parent ? (
            <div>
              <Label>{t('taskDetail.parent')}</Label>
              <Text fz={13} c={tokens.accentTag} fw={600} mt={4}>
                {parent.title}
              </Text>
            </div>
          ) : null}

          {task.subtasks && task.subtasks.length > 0 ? (
            <div>
              <Label>{t('taskDetail.subtasks')}</Label>
              <Stack gap={4} mt={4}>
                {task.subtasks.map((sub) => {
                  const subStatus = statusById.get(sub.statusId);
                  return (
                    <Group key={sub.id} gap={7} wrap="nowrap">
                      <span
                        style={{
                          width: 7,
                          height: 7,
                          borderRadius: 999,
                          flex: '0 0 auto',
                          background: statusColor(
                            subStatus?.color,
                            statuses.findIndex((item) => item.id === sub.statusId),
                          ),
                        }}
                      />
                      <Text fz={13} c={tokens.text} truncate>
                        {sub.title}
                      </Text>
                    </Group>
                  );
                })}
              </Stack>
            </div>
          ) : null}

          <Divider />

          <AttachmentSection ownerType="task" ownerId={task.id} />
        </Stack>
      ) : null}
    </Modal>
  );
}

export default TaskDetailModal;
