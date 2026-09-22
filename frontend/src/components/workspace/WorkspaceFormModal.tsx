import { useEffect, useState } from 'react';
import { Button, Group, Modal, Stack, Text, TextInput, Textarea } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { useSession, useWorkspace } from '../../context';
import { tokens } from '../../theme';
import type { Workspace } from '../../types';

export type WorkspaceFormMode = 'create' | 'edit';

interface WorkspaceFormModalProps {
  opened: boolean;
  mode: WorkspaceFormMode;
  onClose: () => void;
  workspace: Workspace | null;
}

export function WorkspaceFormModal({ opened, mode, onClose, workspace }: WorkspaceFormModalProps) {
  const { t } = useTranslation();
  const { profileId } = useSession();
  const { workspaces, create, update } = useWorkspace();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const isEdit = mode === 'edit' && workspace !== null;

  useEffect(() => {
    if (!opened) {
      return;
    }
    if (workspace) {
      setName(workspace.name);
      setDescription(workspace.description ?? '');
      return;
    }
    setName(t('workspace.defaultName', { n: workspaces.length + 1 }));
    setDescription('');
  }, [opened, workspace, workspaces.length, t]);

  const submit = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      return;
    }
    setSubmitting(true);
    try {
      if (isEdit && workspace) {
        await update(workspace.id, { name: trimmed, description: description.trim() });
        notifications.show({ title: t('workspace.updated'), message: trimmed, color: 'teal' });
      } else {
        if (!profileId) {
          throw new Error(t('errors.noProfile'));
        }
        await create(profileId, trimmed, description.trim());
        notifications.show({ title: t('workspace.created'), message: trimmed, color: 'teal' });
      }
      onClose();
    } catch (cause) {
      notifications.show({
        title: isEdit ? t('workspace.updateFailed') : t('workspace.createFailed'),
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
          {isEdit ? t('workspace.formTitle') : t('workspace.createTitle')}
        </Text>
      }
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label={t('workspace.nameLabel')}
          placeholder={t('workspace.namePlaceholder')}
          value={name}
          onChange={(event) => setName(event.currentTarget.value)}
          required
          data-autofocus
        />
        <Textarea
          label={t('workspace.descriptionLabel')}
          placeholder={t('workspace.descriptionPlaceholder')}
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
          autosize
          minRows={2}
          maxRows={4}
        />
        <Group justify="flex-end" mt="sm">
          <Button variant="default" onClick={onClose} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button onClick={submit} loading={submitting} disabled={!name.trim()}>
            {isEdit ? t('workspace.save') : t('workspace.createSubmit')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

export default WorkspaceFormModal;
