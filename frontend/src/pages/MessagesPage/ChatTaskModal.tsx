import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import {
  ActionIcon,
  Box,
  Button,
  FileButton,
  Group,
  Modal,
  SegmentedControl,
  Stack,
  Text,
  TextInput,
  Textarea,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconFile, IconPaperclip, IconX } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { formatBytes } from '../../lib/format';
import { fileKey, mergeFiles } from '../../lib/taskAttachments';
import { tokens } from '../../theme';
import type { ChatTaskDraft, ChatTaskPriority } from '../../types';
import classes from './MessagesPage.module.css';

const MAX_TITLE = 200;
const MAX_DESCRIPTION = 4000;
/** Số tệp tối đa đính kèm một lần giao task. */
export const MAX_TASK_FILES = 5;

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
  /** Gửi task rồi tải các tệp đã chọn lên task vừa tạo. */
  onSubmit: (draft: ChatTaskDraft, files: File[]) => Promise<unknown>;
}) {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<ChatTaskPriority>('medium');
  const [due, setDue] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [saving, setSaving] = useState(false);
  const resetFilesRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (opened) {
      setTitle(initialTitle.slice(0, MAX_TITLE));
      setDescription('');
      setPriority('medium');
      setDue('');
      setFiles([]);
    }
  }, [opened, initialTitle]);

  const addFiles = (picked: File[]) => {
    if (picked.length === 0) {
      return;
    }
    const merged = mergeFiles(files, picked, MAX_TASK_FILES);
    // Có tệp bị bỏ vì đã đủ giới hạn (không tính tệp trùng).
    const dropped = mergeFiles(files, picked, Number.MAX_SAFE_INTEGER).length > merged.length;
    if (dropped) {
      notifications.show({
        title: t('chatTask.tooManyFiles', { max: MAX_TASK_FILES }),
        message: t('chatTask.tooManyFilesHint'),
        color: 'yellow',
      });
    }
    setFiles(merged);
    resetFilesRef.current?.();
  };

  const removeFile = (key: string) => {
    setFiles((current) => current.filter((file) => fileKey(file) !== key));
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim() || saving) {
      return;
    }
    setSaving(true);
    try {
      await onSubmit({ title: title.trim(), description: description.trim(), priority, dueAt: toIso(due) }, files);
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
          <div>
            <Group justify="space-between" wrap="nowrap" mb={4}>
              <Text fz={14} fw={500}>
                {t('chatTask.fieldFiles')}
              </Text>
              <FileButton resetRef={resetFilesRef} multiple onChange={addFiles}>
                {(props) => (
                  <Button
                    {...props}
                    size="compact-xs"
                    variant="light"
                    leftSection={<IconPaperclip size={13} />}
                    disabled={saving || files.length >= MAX_TASK_FILES}
                  >
                    {t('chatTask.addFiles')}
                  </Button>
                )}
              </FileButton>
            </Group>
            {files.length === 0 ? (
              <Text fz={12} c={tokens.textFaint}>
                {t('chatTask.filesHint', { name: peerName, max: MAX_TASK_FILES })}
              </Text>
            ) : (
              <Stack gap={4}>
                {files.map((file) => {
                  const key = fileKey(file);
                  return (
                    <Group key={key} className={classes.pickedFile} justify="space-between" wrap="nowrap" gap="xs">
                      <Group gap={8} wrap="nowrap" style={{ minWidth: 0 }}>
                        <IconFile size={15} style={{ color: tokens.textMuted, flex: '0 0 auto' }} />
                        <Box style={{ minWidth: 0 }}>
                          <Text fz={12.5} fw={600} c={tokens.text} truncate>
                            {file.name}
                          </Text>
                          <Text fz={11} c={tokens.textFaint}>
                            {formatBytes(file.size)}
                          </Text>
                        </Box>
                      </Group>
                      <ActionIcon
                        variant="subtle"
                        color="gray"
                        size="sm"
                        disabled={saving}
                        onClick={() => removeFile(key)}
                        aria-label={t('chatTask.removeFile', { name: file.name })}
                      >
                        <IconX size={14} />
                      </ActionIcon>
                    </Group>
                  );
                })}
              </Stack>
            )}
          </div>
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
