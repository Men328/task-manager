import { Badge, Button, Divider, Group, Modal, Stack, Text } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { updateTask } from '../../api/task';
import { useBoard, useProject } from '../../context';
import { derivedProgress, formatDateTime, formatShortDate, hasText } from '../../lib/format';
import { DEFAULT_CUSTOM, browserTimeZone, presetOf as repeatPresetOf, ruleFor } from '../../lib/recurrence';
import type { RepeatCustom, RepeatPreset } from '../../lib/recurrence';
import { presetOf, reminderFor, toLocalInput } from '../../lib/reminders';
import type { ReminderPreset } from '../../lib/reminders';
import { statusColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import ProjectSelect from '../common/ProjectSelect';
import { useProjectMoveNotice } from '../common/useProjectMoveNotice';
import AttachmentSection from '../attachment/AttachmentSection';
import PriorityBadge from '../board/PriorityBadge';
import TaskProgress from '../board/TaskProgress';
import RepeatBadge from './RepeatBadge';
import ReminderField from './ReminderField';
import RepeatField from './RepeatField';

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
  const { statusById, statuses, taskById, isStatusDone, moveToProject, refresh } = useBoard();
  const { currentProjectId } = useProject();
  const notifyProjectMove = useProjectMoveNotice();
  const [moving, setMoving] = useState(false);
  const [savedRemindAt, setSavedRemindAt] = useState<string | null>(null);
  const [reminderPreset, setReminderPreset] = useState<ReminderPreset>('none');
  const [reminderCustom, setReminderCustom] = useState('');
  const [savingReminder, setSavingReminder] = useState(false);
  const [savedRule, setSavedRule] = useState('');
  const [repeatPreset, setRepeatPreset] = useState<RepeatPreset>('none');
  const [repeatCustom, setRepeatCustom] = useState<RepeatCustom>(DEFAULT_CUSTOM);
  const [savingRepeat, setSavingRepeat] = useState(false);

  useEffect(() => {
    if (!opened || !task) {
      return;
    }
    const rule = task.recurrenceRule ?? '';
    const repeat = repeatPresetOf(rule, task.dueAt);
    setSavedRule(rule);
    setRepeatPreset(repeat.preset);
    setRepeatCustom(repeat.custom);
  }, [opened, task?.id, task?.recurrenceRule, task?.dueAt]);

  const draftRule = task ? ruleFor(repeatPreset, task.dueAt, repeatCustom) ?? '' : '';
  const repeatDirty = draftRule !== savedRule;

  const saveRepeat = async () => {
    if (!task) {
      return;
    }
    setSavingRepeat(true);
    try {
      await updateTask(task.id, { recurrenceRule: draftRule, recurrenceTz: browserTimeZone() });
      setSavedRule(draftRule);
      refresh();
      notifications.show({ message: draftRule ? t('repeat.saved') : t('repeat.cleared'), color: 'teal' });
    } catch (cause) {
      notifications.show({ title: t('repeat.saveFailed'), message: getErrorMessage(cause), color: 'red' });
    } finally {
      setSavingRepeat(false);
    }
  };

  useEffect(() => {
    if (!opened || !task) {
      return;
    }
    const current = hasText(task.remindAt) ? new Date(task.remindAt as string) : null;
    setSavedRemindAt(current ? current.toISOString() : null);
    setReminderPreset(presetOf(current, task.dueAt));
    setReminderCustom(current ? toLocalInput(current) : '');
  }, [opened, task?.id, task?.remindAt, task?.dueAt]);

  const draftRemindAt = task ? reminderFor(reminderPreset, task.dueAt, reminderCustom)?.toISOString() ?? null : null;
  const reminderDirty = draftRemindAt !== savedRemindAt && !(reminderPreset === 'custom' && !draftRemindAt);

  const saveReminder = async () => {
    if (!task) {
      return;
    }
    setSavingReminder(true);
    try {
      await updateTask(task.id, { remindAt: draftRemindAt });
      setSavedRemindAt(draftRemindAt);
      refresh();
      notifications.show({
        message: draftRemindAt ? t('reminder.saved') : t('reminder.cleared'),
        color: 'teal',
      });
    } catch (cause) {
      notifications.show({ title: t('reminder.saveFailed'), message: getErrorMessage(cause), color: 'red' });
    } finally {
      setSavingReminder(false);
    }
  };

  const status = task ? statusById.get(task.statusId) : undefined;
  const statusIndex = status ? statuses.findIndex((item) => item.id === status.id) : -1;
  const parent = task && hasText(task.parentTaskId)
    ? taskById.get(task.parentTaskId as string)
    : undefined;
  const progress = task ? derivedProgress(task.subtasks, isStatusDone) : null;

  const handleProjectChange = async (nextId: string) => {
    if (!task || nextId === task.projectId || moving) {
      return;
    }
    setMoving(true);
    try {
      await moveToProject(task.id, nextId);
      onClose();
      if (nextId === currentProjectId) {
        notifications.show({ message: t('project.moved'), color: 'teal' });
      } else {
        notifyProjectMove(nextId);
      }
    } catch (cause) {
      notifications.show({
        title: t('project.moveFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setMoving(false);
    }
  };

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
              <RepeatBadge rule={task.recurrenceRule} />
            </Group>
          </div>

          <Divider />

          <ProjectSelect
            value={task.projectId}
            onChange={(id) => void handleProjectChange(id)}
            disabled={moving || hasText(task.parentTaskId)}
            disabledReason={hasText(task.parentTaskId) ? t('project.subtaskLocked') : undefined}
          />

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

          <div>
            <RepeatField
              preset={repeatPreset}
              custom={repeatCustom}
              due={task.dueAt}
              disabled={savingRepeat}
              onChange={(preset, custom) => {
                setRepeatPreset(preset);
                setRepeatCustom(custom);
              }}
            />
            {repeatDirty ? (
              <Group justify="flex-end" mt={6}>
                <Button size="compact-sm" onClick={() => void saveRepeat()} loading={savingRepeat}>
                  {t('repeat.save')}
                </Button>
              </Group>
            ) : null}
          </div>

          <div>
            <ReminderField
              preset={reminderPreset}
              custom={reminderCustom}
              due={task.dueAt}
              disabled={savingReminder}
              onChange={(preset, custom) => {
                setReminderPreset(preset);
                setReminderCustom(custom);
              }}
            />
            {reminderDirty ? (
              <Group justify="flex-end" mt={6}>
                <Button size="compact-sm" onClick={() => void saveReminder()} loading={savingReminder}>
                  {t('reminder.save')}
                </Button>
              </Group>
            ) : null}
          </div>

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
