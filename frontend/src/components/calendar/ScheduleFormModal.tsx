import { useEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  ColorInput,
  Group,
  Modal,
  Stack,
  Text,
  Textarea,
  TextInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import AttachmentSection from '../attachment/AttachmentSection';
import { useCalendar } from '../../context';
import type { NewScheduleInput } from '../../context/calendar/useCalendar';
import type { Schedule } from '../../types';
import { tokens } from '../../theme';

const DEFAULT_COLOR = '#4353e8';

interface ScheduleFormModalProps {
  opened: boolean;
  onClose: () => void;
  /** null = tạo mới; có giá trị = sửa lịch đó. */
  schedule: Schedule | null;
  /** Thời điểm bắt đầu gợi ý khi tạo mới (ISO). */
  defaultStart?: string | null;
  defaultAllDay?: boolean;
  onDelete?: (schedule: Schedule) => void;
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

/** ISO -> value cho input date/datetime-local (theo giờ local). */
function toInput(iso: string, allDay: boolean, end: boolean): string {
  if (!iso) {
    return '';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  if (allDay) {
    // end của all-day lưu dạng exclusive (FullCalendar) -> hiển thị inclusive.
    if (end) {
      date.setDate(date.getDate() - 1);
    }
    return localDate(date);
  }
  return localDateTime(date);
}

export function ScheduleFormModal({
  opened,
  onClose,
  schedule,
  defaultStart,
  defaultAllDay,
  onDelete,
  deleting,
}: ScheduleFormModalProps) {
  const { t } = useTranslation();
  const { create, update } = useCalendar();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [allDay, setAllDay] = useState(false);
  const [startAt, setStartAt] = useState('');
  const [endAt, setEndAt] = useState('');
  const [color, setColor] = useState(DEFAULT_COLOR);
  const [submitting, setSubmitting] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  useEffect(() => {
    if (!opened) {
      return;
    }
    setConfirmingDelete(false);
    if (schedule) {
      setTitle(schedule.title);
      setDescription(schedule.description ?? '');
      setLocation(schedule.location ?? '');
      setAllDay(schedule.allDay);
      setStartAt(schedule.startAt);
      setEndAt(schedule.endAt ?? '');
      setColor(schedule.color || DEFAULT_COLOR);
      return;
    }
    setTitle('');
    setDescription('');
    setLocation('');
    setAllDay(defaultAllDay ?? false);
    setStartAt(defaultStart ?? new Date().toISOString());
    setEndAt('');
    setColor(DEFAULT_COLOR);
  }, [opened, schedule, defaultStart, defaultAllDay]);

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
    if (allDay) {
      const date = new Date(`${value}T00:00:00`);
      date.setDate(date.getDate() + 1);
      setEndAt(date.toISOString());
      return;
    }
    setEndAt(new Date(value).toISOString());
  };

  const handleAllDayChange = (checked: boolean) => {
    setAllDay(checked);
    // Đổi kiểu input làm giá trị end mơ hồ (date <-> datetime) -> yêu cầu nhập lại.
    setEndAt('');
  };

  const handleSubmit = async () => {
    if (!title.trim()) {
      notifications.show({
        title: t('scheduleForm.saveFailed'),
        message: t('scheduleForm.titleRequired'),
        color: 'red',
      });
      return;
    }
    if (!startAt) {
      notifications.show({
        title: t('scheduleForm.saveFailed'),
        message: t('scheduleForm.startRequired'),
        color: 'red',
      });
      return;
    }

    const payload: NewScheduleInput = {
      title: title.trim(),
      description,
      location,
      startAt,
      endAt: endAt || undefined,
      allDay,
      color,
    };

    setSubmitting(true);
    try {
      if (schedule) {
        await update(schedule.id, payload);
      } else {
        await create(payload);
      }
      notifications.show({
        title: schedule ? t('scheduleForm.updated') : t('scheduleForm.created'),
        message: title.trim(),
        color: 'teal',
      });
      onClose();
    } catch (cause) {
      notifications.show({
        title: t('scheduleForm.saveFailed'),
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
      title={schedule ? t('scheduleForm.editTitle') : t('scheduleForm.title')}
      size="md"
    >
      <Stack gap="sm">
        <TextInput
          label={t('scheduleForm.titleLabel')}
          placeholder={t('scheduleForm.titlePlaceholder')}
          value={title}
          onChange={(event) => setTitle(event.currentTarget.value)}
          data-autofocus
          required
        />

        <Checkbox
          label={t('scheduleForm.allDayLabel')}
          checked={allDay}
          onChange={(event) => handleAllDayChange(event.currentTarget.checked)}
        />

        <Group grow align="flex-start">
          <TextInput
            type={allDay ? 'date' : 'datetime-local'}
            label={t('scheduleForm.startLabel')}
            value={toInput(startAt, allDay, false)}
            onChange={(event) => handleStartChange(event.currentTarget.value)}
          />
          <TextInput
            type={allDay ? 'date' : 'datetime-local'}
            label={t('scheduleForm.endLabel')}
            value={toInput(endAt, allDay, true)}
            onChange={(event) => handleEndChange(event.currentTarget.value)}
          />
        </Group>

        <TextInput
          label={t('scheduleForm.locationLabel')}
          placeholder={t('scheduleForm.locationPlaceholder')}
          value={location}
          onChange={(event) => setLocation(event.currentTarget.value)}
        />

        <Textarea
          label={t('scheduleForm.descriptionLabel')}
          placeholder={t('scheduleForm.descriptionPlaceholder')}
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
          autosize
          minRows={2}
        />

        <ColorInput
          label={t('scheduleForm.colorLabel')}
          format="hex"
          value={color}
          onChange={setColor}
          swatches={['#4353e8', '#12b886', '#f59f00', '#e8590c', '#c026d3', '#0c8599']}
        />

        {schedule?.id ? (
          <AttachmentSection ownerType="schedule" ownerId={schedule.id} />
        ) : null}

        {confirmingDelete ? (
          <Stack gap={6}>
            <Text fz={13} c={tokens.text}>
              {t('scheduleForm.deleteConfirm')}
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
                  if (schedule) {
                    onDelete?.(schedule);
                  }
                }}
              >
                {t('common.remove')}
              </Button>
            </Group>
          </Stack>
        ) : (
          <Group justify="space-between" mt={4}>
            {schedule && onDelete ? (
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
                {t('scheduleForm.submit')}
              </Button>
            </Group>
          </Group>
        )}
      </Stack>
    </Modal>
  );
}

export default ScheduleFormModal;
