import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { Button, Group, Modal, SegmentedControl, Stack, Text, TextInput, Textarea } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { tokens } from '../../theme';
import type { ChatTaskDraft, ChatTaskPriority } from '../../types';

const MAX_TITLE = 200;
const MAX_DESCRIPTION = 4000;

function toIso(local: string): string {
  if (!local) {
    return '';
  }
  const date = new Date(local);
  return Number.isNaN(date.getTime()) ? '' : date.toISOString();
}

export function ChatTaskModal({
  opened,
  peerName,
  initialTitle,
  onClose,
  onSubmit,
}: {
  opened: boolean;
  peerName: string;
  initialTitle: string;
  onClose: () => void;
  onSubmit: (draft: ChatTaskDraft) => Promise<unknown>;
}) {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<ChatTaskPriority>('medium');
  const [due, setDue] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (opened) {
      setTitle(initialTitle.slice(0, MAX_TITLE));
      setDescription('');
      setPriority('medium');
      setDue('');
    }
  }, [opened, initialTitle]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim() || saving) {
      return;
    }
    setSaving(true);
    try {
      await onSubmit({ title: title.trim(), description: description.trim(), priority, dueAt: toIso(due) });
      onClose();
    } catch (cause) {
      notifications.show({ title: t('chatTask.failed'), message: getErrorMessage(cause), color: 'red' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={t('chatTask.title', { name: peerName })} centered>
      <form onSubmit={(event) => void submit(event)}>
        <Stack gap="sm">
          <Text fz={12.5} c={tokens.textMuted}>
            {t('chatTask.hint', { name: peerName })}
          </Text>
          <TextInput
            label={t('chatTask.fieldTitle')}
            required
            data-autofocus
            maxLength={MAX_TITLE}
            value={title}
            onChange={(event) => setTitle(event.currentTarget.value)}
          />
          <Textarea
            label={t('chatTask.fieldDescription')}
            autosize
            minRows={2}
            maxRows={6}
            maxLength={MAX_DESCRIPTION}
            value={description}
            onChange={(event) => setDescription(event.currentTarget.value)}
          />
          <div>
            <Text fz={14} fw={500} mb={4}>
              {t('chatTask.fieldPriority')}
            </Text>
            <SegmentedControl
              fullWidth
              value={priority}
              onChange={(value) => setPriority(value as ChatTaskPriority)}
              data={(['low', 'medium', 'high', 'urgent'] as const).map((value) => ({
                value,
                label: t(`chatTask.priority.${value}`),
              }))}
            />
          </div>
          <TextInput
            type="datetime-local"
            label={t('chatTask.fieldDue')}
            value={due}
            onChange={(event) => setDue(event.currentTarget.value)}
          />
          <Group justify="flex-end" mt="xs">
            <Button variant="default" onClick={onClose} disabled={saving}>
              {t('chatTask.cancel')}
            </Button>
            <Button type="submit" loading={saving} disabled={!title.trim()}>
              {t('chatTask.submit')}
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
