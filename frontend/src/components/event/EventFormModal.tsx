import { useEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  ColorInput,
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
import { useEvent } from '../../context';
import type { NewEventInput } from '../../context/event/useEvent';
import { EVENT_STATUS_LABEL_KEY } from '../../lib/tokens';
import { tokens } from '../../theme';
import { EVENT_STATUSES, type Event, type EventStatus } from '../../types';

const DEFAULT_COLOR = '#4353e8';

interface EventFormModalProps {
  opened: boolean;
  onClose: () => void;
  event: Event | null;
  onDelete?: (event: Event) => void;
  deleting?: boolean;
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

function localDate(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function localDateTime(date: Date): string {
  return `${localDate(date)}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function toInput(iso: string, allDay: boolean): string {
  if (!iso) {
    return '';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return allDay ? localDate(date) : localDateTime(date);
}

export function EventFormModal({ opened, onClose, event, onDelete, deleting }: EventFormModalProps) {
  const { t } = useTranslation();
  const { create, update } = useEvent();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [allDay, setAllDay] = useState(false);
  const [startAt, setStartAt] = useState('');
  const [endAt, setEndAt] = useState('');
  const [color, setColor] = useState(DEFAULT_COLOR);
  const [status, setStatus] = useState<EventStatus>('EVENT_STATUS_PLANNED');
  const [submitting, setSubmitting] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  useEffect(() => {
    if (!opened) {
      return;
    }
    setConfirmingDelete(false);
    if (event) {
      setTitle(event.title);
      setDescription(event.description ?? '');
      setLocation(event.location ?? '');
      setAllDay(event.allDay);
      setStartAt(event.startAt);
      setEndAt(event.endAt ?? '');
      setColor(event.color || DEFAULT_COLOR);
      setStatus(event.status);
      return;
    }
    setTitle('');
    setDescription('');
    setLocation('');
    setAllDay(false);
    setStartAt(new Date().toISOString());
    setEndAt('');
    setColor(DEFAULT_COLOR);
    setStatus('EVENT_STATUS_PLANNED');
  }, [opened, event]);

  const handleStartChange = (value: string) => {
    if (!value) {
      setStartAt('');
      return;
    }
    setStartAt(allDay ? new Date(`${value}T00:00:00`).toISOString() : new Date(value).toISOString());
  };

  const handleEndChange = (value: string) => {
    if (!value) {
      setEndAt('');
      return;
    }
    setEndAt(allDay ? new Date(`${value}T00:00:00`).toISOString() : new Date(value).toISOString());
  };

  const handleSubmit = async () => {
    if (!title.trim()) {
      notifications.show({
        title: t('eventForm.saveFailed'),
        message: t('eventForm.titleRequired'),
        color: 'red',
      });
      return;
    }
    if (!startAt) {
      notifications.show({
        title: t('eventForm.saveFailed'),
        message: t('eventForm.startRequired'),
        color: 'red',
      });
      return;
    }

    const payload: NewEventInput = {
      title: title.trim(),
      description,
      location,
      startAt,
      endAt: endAt || undefined,
      allDay,
      color,
      status,
    };

    setSubmitting(true);
    try {
      if (event) {
        await update(event.id, payload);
      } else {
        await create(payload);
      }
      notifications.show({
        title: event ? t('eventForm.updated') : t('eventForm.created'),
        message: title.trim(),
        color: 'teal',
      });
      onClose();
    } catch (cause) {
      notifications.show({
        title: t('eventForm.saveFailed'),
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
      title={event ? t('eventForm.editTitle') : t('eventForm.title')}
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label={t('eventForm.titleLabel')}
          placeholder={t('eventForm.titlePlaceholder')}
          value={title}
          onChange={(input) => setTitle(input.currentTarget.value)}
          data-autofocus
          required
        />

        <Group grow align="flex-start">
          <Select
            label={t('eventForm.statusLabel')}
            data={EVENT_STATUSES.map((value) => ({ value, label: t(EVENT_STATUS_LABEL_KEY[value]) }))}
            value={status}
            onChange={(value) => setStatus((value as EventStatus) ?? 'EVENT_STATUS_PLANNED')}
            allowDeselect={false}
          />
          <Checkbox
            mt={26}
            label={t('eventForm.allDayLabel')}
            checked={allDay}
            onChange={(input) => {
              setAllDay(input.currentTarget.checked);
              setEndAt('');
            }}
          />
        </Group>

        <Group grow align="flex-start">
          <TextInput
            type={allDay ? 'date' : 'datetime-local'}
            label={t('eventForm.startLabel')}
            value={toInput(startAt, allDay)}
            onChange={(input) => handleStartChange(input.currentTarget.value)}
          />
          <TextInput
            type={allDay ? 'date' : 'datetime-local'}
            label={t('eventForm.endLabel')}
            value={toInput(endAt, allDay)}
            onChange={(input) => handleEndChange(input.currentTarget.value)}
          />
        </Group>

        <TextInput
          label={t('eventForm.locationLabel')}
          placeholder={t('eventForm.locationPlaceholder')}
          value={location}
          onChange={(input) => setLocation(input.currentTarget.value)}
        />

        <Textarea
          label={t('eventForm.descriptionLabel')}
          placeholder={t('eventForm.descriptionPlaceholder')}
          value={description}
          onChange={(input) => setDescription(input.currentTarget.value)}
          autosize
          minRows={2}
        />

        <ColorInput
          label={t('eventForm.colorLabel')}
          format="hex"
          value={color}
          onChange={setColor}
          swatches={['#4353e8', '#12b886', '#f59f00', '#e8590c', '#c026d3', '#0c8599']}
        />

        {event?.source ? (
          <Text fz={12} c={tokens.textMuted}>
            {t('eventForm.sourceLabel')}: {event.source}
          </Text>
        ) : null}

        {event?.id ? <AttachmentSection ownerType="event" ownerId={event.id} /> : null}

        {confirmingDelete ? (
          <Stack gap={6}>
            <Text fz={13} c={tokens.text}>
              {t('eventForm.deleteConfirm')}
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
                  if (event) {
                    onDelete?.(event);
                  }
                }}
              >
                {t('common.remove')}
              </Button>
            </Group>
          </Stack>
        ) : (
          <Group justify="space-between" mt={4}>
            {event && onDelete ? (
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
                {t('eventForm.submit')}
              </Button>
            </Group>
          </Group>
        )}
      </Stack>
    </Modal>
  );
}

export default EventFormModal;
