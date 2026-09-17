import { useState } from 'react';
import {
  ActionIcon,
  Box,
  Button,
  Group,
  Modal,
  Table,
  Text,
  Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconTrash } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { useBoard } from '../../context';
import { derivedProgress, formatShortDate, hasText } from '../../lib/format';
import { statusColor } from '../../lib/tokens';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import PriorityBadge from './PriorityBadge';
import TaskProgress from './TaskProgress';
import classes from './TaskTable.module.css';

/** View dạng bảng — có đầy đủ CRUD (kanban chỉ kéo thả + tạo). */
export function TaskTable() {
  const { t, i18n } = useTranslation();
  const { tasks, statusById, statuses, isStatusDone, taskById, removeTask } = useBoard();
  const [pendingDelete, setPendingDelete] = useState<Task | null>(null);
  const [deleting, setDeleting] = useState(false);

  const confirmDelete = async () => {
    if (!pendingDelete) {
      return;
    }
    setDeleting(true);
    try {
      await removeTask(pendingDelete.id);
      notifications.show({
        title: t('taskTable.deleted'),
        message: pendingDelete.title,
        color: 'teal',
      });
      setPendingDelete(null);
    } catch (cause) {
      notifications.show({
        title: t('taskTable.deleteFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setDeleting(false);
    }
  };

  return (
    <Box className={`${scrollClasses.scroll} ${classes.root}`} p="lg">
      <Box className={classes.panel}>
        <Table highlightOnHover verticalSpacing="sm" horizontalSpacing="md">
          <Table.Thead>
            <Table.Tr className={classes.headRow}>
              <Table.Th>{t('taskTable.colTask')}</Table.Th>
              <Table.Th w={150}>{t('taskTable.colStatus')}</Table.Th>
              <Table.Th w={130}>{t('taskTable.colPriority')}</Table.Th>
              <Table.Th w={170}>{t('taskTable.colProgress')}</Table.Th>
              <Table.Th w={100}>{t('taskTable.colDue')}</Table.Th>
              <Table.Th w={60} />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {tasks.length === 0 ? (
              <Table.Tr>
                <Table.Td colSpan={6}>
                  <Text c={tokens.textMuted} ta="center" py="lg" fz={13}>
                    {t('taskTable.empty')}
                  </Text>
                </Table.Td>
              </Table.Tr>
            ) : (
              tasks.map((task) => {
                const status = statusById.get(task.statusId);
                const statusIndex = statuses.findIndex((item) => item.id === task.statusId);
                const parent = hasText(task.parentTaskId)
                  ? taskById.get(task.parentTaskId as string)
                  : undefined;
                const progress = derivedProgress(task.subtasks, isStatusDone);

                return (
                  <Table.Tr key={task.id}>
                    <Table.Td>
                      <Text fz={13.5} fw={600} c={tokens.text}>
                        {task.title}
                      </Text>
                      {parent ? (
                        <Text fz={11.5} c={tokens.accentTag} fw={600}>
                          {t('taskTable.inParent', { title: parent.title })}
                        </Text>
                      ) : null}
                      {hasText(task.description) ? (
                        <Text fz={11.5} c={tokens.textMuted} lineClamp={1}>
                          {task.description}
                        </Text>
                      ) : null}
                    </Table.Td>
                    <Table.Td>
                      <Group gap={7} wrap="nowrap">
                        <Box
                          className={classes.dot}
                          style={{
                            background: statusColor(status?.color, statusIndex < 0 ? 0 : statusIndex),
                          }}
                        />
                        <Text fz={12.5} fw={600} c={tokens.text}>
                          {status?.name ?? '—'}
                        </Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <PriorityBadge priority={task.priority} />
                    </Table.Td>
                    <Table.Td>
                      {progress === null ? (
                        <Text fz={12} c={tokens.textFaint}>
                          —
                        </Text>
                      ) : (
                        <TaskProgress value={progress} />
                      )}
                    </Table.Td>
                    <Table.Td>
                      <Text fz={12.5} c={tokens.textMuted}>
                        {formatShortDate(task.dueAt, i18n.language) ?? '—'}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Tooltip label={t('taskTable.remove')} withArrow openDelay={300}>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          size="sm"
                          onClick={() => setPendingDelete(task)}
                          aria-label={t('taskTable.removeAria', { title: task.title })}
                        >
                          <IconTrash size={15} />
                        </ActionIcon>
                      </Tooltip>
                    </Table.Td>
                  </Table.Tr>
                );
              })
            )}
          </Table.Tbody>
        </Table>
      </Box>

      <Modal
        opened={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        title={t('taskTable.confirmTitle')}
      >
        <Text fz={13.5} c={tokens.textMuted}>
          {t('taskTable.confirmPrefix')} <b>{pendingDelete?.title}</b>{' '}
          {t('taskTable.confirmSuffix')}
        </Text>
        <Group justify="flex-end" mt="lg">
          <Button variant="default" onClick={() => setPendingDelete(null)} disabled={deleting}>
            {t('common.cancel')}
          </Button>
          <Button color="red" loading={deleting} onClick={confirmDelete}>
            {t('common.remove')}
          </Button>
        </Group>
      </Modal>
    </Box>
  );
}

export default TaskTable;
