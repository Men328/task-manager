import { useMemo, useState } from 'react';
import { ActionIcon, Box, Button, Group, Text, TextInput, Tooltip } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconAdjustmentsHorizontal, IconAlertTriangle, IconBolt, IconRepeat, IconX } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { createTask } from '../../api/task';
import { useProject, useSession } from '../../context';
import { parseQuickAdd } from '../../lib/quickAdd/parse';
import { browserTimeZone, describeRule } from '../../lib/recurrence';
import type { QuickAddKind, QuickAddResult } from '../../lib/quickAdd/parse';
import { PRIORITY_LABEL_KEY, projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Task, TaskPriority } from '../../types';
import classes from './QuickAddInput.module.css';

/** Dữ liệu để mở form đầy đủ với các giá trị đã parse (Shift+Enter / nút "Chi tiết"). */
export interface QuickAddPrefill {
  title: string;
  priority?: TaskPriority;
  projectId?: string;
  /** YYYY-MM-DD theo giờ địa phương (form chỉ chọn ngày). */
  dueDate?: string;
  recurrenceRule?: string;
}

interface QuickAddInputProps {
  /** Project dùng khi dòng nhập không có `#project`. */
  defaultProjectId: string | null;
  onCreated?: (task: Task) => void;
  onOpenTask?: (task: Task) => void;
  onOpenForm?: (prefill: QuickAddPrefill) => void;
}

function toDateInput(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/**
 * Ô thêm nhanh một dòng: hiểu ngày/giờ, `!ưu tiên`, `#project` (xem lib/quickAdd/parse),
 * hiện chip xem trước trước khi Enter; bấm × trên chip để giữ chữ đó trong tiêu đề.
 */
export function QuickAddInput({ defaultProjectId, onCreated, onOpenTask, onOpenForm }: QuickAddInputProps) {
  const { t, i18n } = useTranslation();
  const { profileId } = useSession();
  const { activeProjects } = useProject();
  const [value, setValue] = useState('');
  const [ignore, setIgnore] = useState<Set<QuickAddKind>>(new Set());
  const [submitting, setSubmitting] = useState(false);

  const parsed: QuickAddResult = useMemo(
    () => parseQuickAdd(value, { projects: activeProjects, ignore }),
    [value, activeProjects, ignore],
  );
  const projectId = parsed.projectId ?? defaultProjectId ?? undefined;
  const project = activeProjects.find((item) => item.id === projectId);
  const canSubmit = parsed.title.length > 0 && Boolean(projectId) && !submitting;

  const reset = () => {
    setValue('');
    setIgnore(new Set());
  };

  const dismiss = (...kinds: QuickAddKind[]) => setIgnore((prev) => new Set([...prev, ...kinds]));

  const prefill = (): QuickAddPrefill => ({
    title: parsed.title,
    priority: parsed.priority,
    projectId,
    dueDate: parsed.dueAt ? toDateInput(parsed.dueAt) : undefined,
    recurrenceRule: parsed.recurrenceRule,
  });

  const submit = async () => {
    if (!canSubmit || !profileId || !projectId) {
      return;
    }
    setSubmitting(true);
    try {
      const task = await createTask({
        profileId,
        projectId,
        title: parsed.title,
        priority: parsed.priority,
        dueAt: parsed.dueAt?.toISOString(),
        recurrenceRule: parsed.recurrenceRule,
        recurrenceTz: parsed.recurrenceRule ? browserTimeZone() : undefined,
      });
      const noticeId = `quick-add-${task.id}`;
      notifications.show({
        id: noticeId,
        color: 'teal',
        message: (
          <Group justify="space-between" wrap="nowrap" gap="sm">
            <Text fz={13} truncate>
              {t('quickAdd.created', { project: project?.name ?? '' })}
            </Text>
            {onOpenTask ? (
              <Button
                size="compact-xs"
                variant="light"
                color="teal"
                onClick={() => {
                  notifications.hide(noticeId);
                  onOpenTask(task);
                }}
              >
                {t('quickAdd.open')}
              </Button>
            ) : null}
          </Group>
        ),
      });
      reset();
      onCreated?.(task);
    } catch (cause) {
      notifications.show({ title: t('quickAdd.failed'), message: getErrorMessage(cause), color: 'red' });
    } finally {
      setSubmitting(false);
    }
  };

  const openForm = () => {
    onOpenForm?.(prefill());
    reset();
  };

  const dueLabel = parsed.dueAt
    ? [
        parsed.dueAt.toLocaleDateString(i18n.language, { weekday: 'short', day: 'numeric', month: 'numeric' }),
        parsed.hasTime ? parsed.dueAt.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' }) : null,
      ]
        .filter(Boolean)
        .join(' ')
    : null;
  const hasDateToken = parsed.tokens.some((token) => token.kind === 'date');

  return (
    <Box>
      <TextInput
        value={value}
        onChange={(event) => {
          const next = event.currentTarget.value;
          setValue(next);
          if (next.trim() === '') {
            setIgnore(new Set());
          }
        }}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && event.shiftKey && onOpenForm) {
            event.preventDefault();
            openForm();
          } else if (event.key === 'Enter') {
            event.preventDefault();
            void submit();
          } else if (event.key === 'Escape') {
            reset();
          }
        }}
        placeholder={t('quickAdd.placeholder')}
        aria-label={t('quickAdd.placeholder')}
        leftSection={<IconBolt size={16} style={{ color: tokens.brand }} />}
        rightSection={
          onOpenForm ? (
            <Tooltip label={t('quickAdd.details')} withArrow>
              <ActionIcon variant="subtle" color="gray" onClick={openForm} aria-label={t('quickAdd.details')}>
                <IconAdjustmentsHorizontal size={16} />
              </ActionIcon>
            </Tooltip>
          ) : null
        }
        radius="md"
        disabled={submitting}
        classNames={{ input: classes.input }}
      />

      {value.trim() ? (
        <Group gap={6} mt={6} wrap="wrap">
          <Box className={classes.chip}>
            <span className={classes.dot} style={{ background: projectColor(project) }} />
            <Text fz={12} fw={600}>
              {project?.name ?? t('quickAdd.noProject')}
            </Text>
            {parsed.projectId ? <ChipRemove label={t('quickAdd.remove')} onClick={() => dismiss('project')} /> : null}
          </Box>
          {dueLabel ? (
            <Box className={classes.chip}>
              <Text fz={12} fw={600}>
                {t('quickAdd.due', { value: dueLabel })}
              </Text>
              <ChipRemove label={t('quickAdd.remove')} onClick={() => dismiss(hasDateToken ? 'date' : 'time', 'time')} />
            </Box>
          ) : null}
          {parsed.recurrenceRule ? (
            <Box className={classes.chip}>
              <IconRepeat size={13} />
              <Text fz={12} fw={600}>
                {describeRule(parsed.recurrenceRule, t, i18n.language)}
              </Text>
              <ChipRemove label={t('quickAdd.remove')} onClick={() => dismiss('repeat')} />
            </Box>
          ) : null}
          {parsed.priority ? (
            <Box className={classes.chip}>
              <Text fz={12} fw={600}>
                {t(PRIORITY_LABEL_KEY[parsed.priority])}
              </Text>
              <ChipRemove label={t('quickAdd.remove')} onClick={() => dismiss('priority')} />
            </Box>
          ) : null}
          {parsed.warnings.map((warning) => (
            <Box key={`${warning.kind}-${warning.value}`} className={classes.chip} data-warning>
              <IconAlertTriangle size={13} />
              <Text fz={12} fw={600}>
                {t(`quickAdd.${warning.kind}`, { value: warning.value })}
              </Text>
            </Box>
          ))}
          {parsed.title ? null : (
            <Text fz={12} c={tokens.textMuted}>
              {t('quickAdd.needTitle')}
            </Text>
          )}
        </Group>
      ) : null}
    </Box>
  );
}

function ChipRemove({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <ActionIcon size={16} variant="subtle" color="gray" radius="xl" onClick={onClick} aria-label={label}>
      <IconX size={11} />
    </ActionIcon>
  );
}

export default QuickAddInput;
