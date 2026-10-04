import { useEffect, useState } from 'react';
import {
  Badge,
  Button,
  Code,
  Group,
  Modal,
  Select,
  Stack,
  Text,
  Textarea,
  TextInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import AttachmentSection from '../attachment/AttachmentSection';
import { useBacklog } from '../../context';
import {
  BACKLOG_CATEGORY_LABEL_KEY,
  BACKLOG_STATUS_LABEL_KEY,
} from '../../lib/tokens';
import { tokens } from '../../theme';
import { BACKLOG_STATUSES, type Backlog, type BacklogStatus } from '../../types';

interface BacklogFormModalProps {
  opened: boolean;
  onClose: () => void;
  item: Backlog | null;
  onDelete?: (item: Backlog) => void;
  deleting?: boolean;
}

export function BacklogFormModal({ opened, onClose, item, onDelete, deleting }: BacklogFormModalProps) {
  const { t } = useTranslation();
  const { create, update } = useBacklog();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [reason, setReason] = useState('');
  const [status, setStatus] = useState<BacklogStatus>('BACKLOG_STATUS_NEW');
  const [submitting, setSubmitting] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  useEffect(() => {
    if (!opened) {
      return;
    }
    setConfirmingDelete(false);
    if (item) {
      setTitle(item.title);
      setDescription(item.description ?? '');
      setReason(item.reason ?? '');
      setStatus(item.status);
      return;
    }
    setTitle('');
    setDescription('');
    setReason('');
    setStatus('BACKLOG_STATUS_NEW');
  }, [opened, item]);

  const handleSubmit = async () => {
    if (!title.trim()) {
      notifications.show({
        title: t('backlogForm.saveFailed'),
        message: t('backlogForm.titleRequired'),
        color: 'red',
      });
      return;
    }

    setSubmitting(true);
    try {
      if (item) {
        await update(item.id, {
          title: title.trim(),
          description,
          reason,
          status,
        });
      } else {
        await create({
          title: title.trim(),
          description,
          reason,
          status,
          category: 'other',
        });
      }
      notifications.show({
        title: item ? t('backlogForm.updated') : t('backlogForm.created'),
        message: title.trim(),
        color: 'teal',
      });
      onClose();
    } catch (cause) {
      notifications.show({
        title: t('backlogForm.saveFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setSubmitting(false);
    }
  };

  const categoryKey = BACKLOG_CATEGORY_LABEL_KEY[item?.category ?? 'other'];

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={item ? t('backlogForm.editTitle') : t('backlogForm.title')}
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label={t('backlogForm.titleLabel')}
          value={title}
          onChange={(input) => setTitle(input.currentTarget.value)}
          data-autofocus
          required
        />

        <Group gap="xs">
          <Badge size="sm" variant="light" color="grape">
            {categoryKey ? t(categoryKey) : (item?.category ?? 'other')}
          </Badge>
          {item?.sender ? (
            <Text fz={12} c={tokens.textMuted}>
              {t('backlogForm.senderLabel')}: {item.sender}
            </Text>
          ) : null}
        </Group>

        <Textarea
          label={t('backlogForm.descriptionLabel')}
          value={description}
          onChange={(input) => setDescription(input.currentTarget.value)}
          autosize
          minRows={3}
        />

        <Textarea
          label={t('backlogForm.reasonLabel')}
          description={t('backlogForm.reasonHint')}
          value={reason}
          onChange={(input) => setReason(input.currentTarget.value)}
          autosize
          minRows={2}
        />

        <Select
          label={t('backlogForm.statusLabel')}
          data={BACKLOG_STATUSES.map((value) => ({
            value,
            label: t(BACKLOG_STATUS_LABEL_KEY[value]),
          }))}
          value={status}
          onChange={(value) => setStatus((value as BacklogStatus) ?? 'BACKLOG_STATUS_NEW')}
          allowDeselect={false}
        />

        {item?.objectKey ? (
          <Stack gap={2}>
            <Text fz={12} c={tokens.textMuted}>
              {t('backlogForm.objectKeyLabel')}
            </Text>
            <Code>{item.objectKey}</Code>
          </Stack>
        ) : null}

        {item?.id ? <AttachmentSection ownerType="backlog" ownerId={item.id} /> : null}

        {confirmingDelete ? (
          <Stack gap={6}>
            <Text fz={13} c={tokens.text}>
              {t('backlogForm.deleteConfirm')}
            </Text>
            <Group justify="flex-end">
              <Button variant="default" size="xs" onClick={() => setConfirmingDelete(false)}>
                {t('common.cancel')}
              </Button>
              <Button
                color="red"
                size="xs"
                loading={deleting}
                onClick={() => {
                  if (item) {
                    onDelete?.(item);
                  }
                }}
              >
                {t('common.remove')}
              </Button>
            </Group>
          </Stack>
        ) : (
          <Group justify="space-between" mt={4}>
            {item && onDelete ? (
              <Button color="red" variant="subtle" onClick={() => setConfirmingDelete(true)}>
                {t('common.remove')}
              </Button>
            ) : (
              <span />
            )}
            <Group>
              <Button variant="default" onClick={onClose}>
                {t('common.cancel')}
              </Button>
              <Button loading={submitting} onClick={handleSubmit}>
                {t('backlogForm.submit')}
              </Button>
            </Group>
          </Group>
        )}
      </Stack>
    </Modal>
  );
}

export default BacklogFormModal;
