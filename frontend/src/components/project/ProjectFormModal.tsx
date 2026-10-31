import { useEffect, useState } from 'react';
import { Button, ColorInput, Group, Modal, Stack, Textarea, TextInput } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { useProject } from '../../context';
import type { Project } from '../../types';

const DEFAULT_COLOR = '#4353e8';
const SWATCHES = ['#4353e8', '#12b886', '#f59f00', '#e8590c', '#c026d3', '#0c8599'];

interface ProjectFormModalProps {
  opened: boolean;
  onClose: () => void;
  /** null = tạo mới; có giá trị = sửa project đó. */
  project: Project | null;
  /** Gọi sau khi tạo mới thành công (để switcher chọn project vừa tạo). */
  onCreated?: (project: Project) => void;
}

export function ProjectFormModal({ opened, onClose, project, onCreated }: ProjectFormModalProps) {
  const { t } = useTranslation();
  const { save } = useProject();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [color, setColor] = useState(DEFAULT_COLOR);
  const [saving, setSaving] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);

  useEffect(() => {
    if (!opened) {
      return;
    }
    setName(project?.name ?? '');
    setDescription(project?.description ?? '');
    setColor(project?.color || DEFAULT_COLOR);
    setNameError(null);
  }, [opened, project]);

  const handleSubmit = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(t('project.nameRequired'));
      return;
    }
    setSaving(true);
    try {
      if (project) {
        await save({ name: trimmed, description, color }, project.id);
      } else {
        const created = await save({ profileId: '', name: trimmed, description, color });
        onCreated?.(created);
      }
      onClose();
    } catch (cause) {
      notifications.show({
        title: t('project.saveFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={project ? t('project.editTitle') : t('project.createTitle')}
      centered
    >
      <Stack gap="sm">
        <TextInput
          label={t('project.nameLabel')}
          placeholder={t('project.namePlaceholder')}
          value={name}
          onChange={(event) => {
            setName(event.currentTarget.value);
            setNameError(null);
          }}
          error={nameError}
          data-autofocus
          withAsterisk
        />
        <Textarea
          label={t('project.descriptionLabel')}
          description={t('project.descriptionHint')}
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
          autosize
          minRows={3}
          maxRows={8}
        />
        <ColorInput
          label={t('project.colorLabel')}
          format="hex"
          value={color}
          onChange={setColor}
          swatches={SWATCHES}
        />
        <Group justify="flex-end" mt={4}>
          <Button variant="default" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button loading={saving} onClick={handleSubmit}>
            {project ? t('common.save') : t('project.create')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

export default ProjectFormModal;
