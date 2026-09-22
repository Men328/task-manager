import { useEffect, useMemo, useState } from 'react';
import { Box, Button, Group, Modal, Select, Stack, Text, TextInput, Textarea } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconCalendar } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { useBoard } from '../../context';
import { hasText } from '../../lib/format';
import { PRIORITY_LABEL_KEY } from '../../lib/tokens';
import { readWorkspaceId } from '../../lib/workspace';
import { tokens } from '../../theme';
import { TASK_PRIORITIES } from '../../types';
import type { TaskPriority } from '../../types';

interface TaskFormModalProps {
  opened: boolean;
  onClose: () => void;
  /** status được chọn sẵn (khi bấm "+ Add Task" ở 1 cột) */
  presetStatusId?: string | null;
}

export function TaskFormModal({ opened, onClose, presetStatusId }: TaskFormModalProps) {
  const { t } = useTranslation();
  const { statuses, tasks, addTask } = useBoard();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [statusId, setStatusId] = useState<string | null>(null);
  const [priority, setPriority] = useState<TaskPriority>('TASK_PRIORITY_MEDIUM');
  const [parentTaskId, setParentTaskId] = useState<string | null>(null);
  const [dueAt, setDueAt] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // reset mỗi lần mở
  useEffect(() => {
    if (!opened) {
      return;
    }
    const fallback = statuses.find((status) => status.isDefault)?.id ?? statuses[0]?.id ?? null;
    setTitle('');
    setDescription('');
    setStatusId(presetStatusId ?? fallback);
    setPriority('TASK_PRIORITY_MEDIUM');
    setParentTaskId(null);
    setDueAt('');
  }, [opened, presetStatusId, statuses]);

  const statusOptions = useMemo(
    () => statuses.map((status) => ({ value: status.id, label: status.name })),
    [statuses],
  );

  const parentOptions = useMemo(
    () => tasks.map((task) => ({ value: task.id, label: task.title })),
    [tasks],
  );

  const priorityOptions = useMemo(
    () =>
      TASK_PRIORITIES.map((value) => ({
        value,
        label: t(PRIORITY_LABEL_KEY[value]),
      })),
    [t],
  );

  const submit = async () => {
    if (!title.trim()) {
      return;
    }
    const workspaceId = readWorkspaceId();
    if (!workspaceId) {
      notifications.show({
        title: t('taskForm.createFailed'),
        message: t('taskForm.noWorkspace'),
        color: 'red',
      });
      return;
    }
    setSubmitting(true);
    try {
      await addTask({
        title: title.trim(),
        workspaceId,
        description: description.trim() || undefined,
        statusId: statusId ?? undefined,
        priority,
        parentTaskId: parentTaskId ?? undefined,
        dueAt: hasText(dueAt) ? new Date(`${dueAt}T00:00:00`).toISOString() : undefined,
      });
      notifications.show({ title: t('taskForm.created'), message: title.trim(), color: 'teal' });
      onClose();
    } catch (cause) {
      notifications.show({
        title: t('taskForm.createFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Text fz={16} fw={800} c={tokens.text}>
          {t('taskForm.title')}
        </Text>
      }
      size="lg"
    >
      <Stack gap="sm">
        <TextInput
          label={t('taskForm.titleLabel')}
          placeholder={t('taskForm.titlePlaceholder')}
          value={title}
          onChange={(event) => setTitle(event.currentTarget.value)}
          required
          data-autofocus
        />

        <Textarea
          label={t('taskForm.descriptionLabel')}
          placeholder={t('taskForm.descriptionPlaceholder')}
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
          autosize
          minRows={2}
          maxRows={4}
        />

        <Group grow align="flex-start">
          <Select
            label={t('taskForm.statusLabel')}
            placeholder={t('taskForm.statusPlaceholder')}
            data={statusOptions}
            value={statusId}
            onChange={setStatusId}
            searchable
            nothingFoundMessage={t('taskForm.noStatus')}
          />
          <Select
            label={t('taskForm.priorityLabel')}
            data={priorityOptions}
            value={priority}
            onChange={(value) => setPriority((value as TaskPriority) ?? 'TASK_PRIORITY_MEDIUM')}
            allowDeselect={false}
          />
        </Group>

        <Group grow align="flex-start">
          <Select
            label={t('taskForm.parentLabel')}
            placeholder={t('taskForm.parentPlaceholder')}
            data={parentOptions}
            value={parentTaskId}
            onChange={setParentTaskId}
            searchable
            clearable
            nothingFoundMessage={t('taskForm.noTasks')}
          />
          <Box>
            <Text fz={13} fw={600} mb={5} c={tokens.text}>
              {t('taskForm.dueLabel')}
            </Text>
            <TextInput
              type="date"
              value={dueAt}
              onChange={(event) => setDueAt(event.currentTarget.value)}
              leftSection={<IconCalendar size={15} style={{ color: tokens.textFaint }} />}
            />
          </Box>
        </Group>

        <Group justify="flex-end" mt="sm">
          <Button variant="default" onClick={onClose} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button onClick={submit} loading={submitting} disabled={!title.trim()}>
            {t('taskForm.submit')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

export default TaskFormModal;
