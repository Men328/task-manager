import { ActionIcon, Box, Group, Text, Tooltip } from '@mantine/core';
import { IconCircle, IconCircleCheck, IconCornerDownRight } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { calendarDaysBetween, isLocalMidnight } from '../../lib/dateRanges';
import { projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Project, Task } from '../../types';
import PriorityBadge from '../board/PriorityBadge';
import classes from './Agenda.module.css';

export type AgendaRowMode = 'overdue' | 'today' | 'upcoming';

interface AgendaRowProps {
  task: Task;
  project: Project | undefined;
  parentTitle?: string;
  mode: AgendaRowMode;
  today0: Date;
  completing: boolean;
  /** null = bấm được; ngược lại là lý do nút "xong" bị khoá (hiện trong tooltip). */
  completeBlockedReason: string | null;
  onComplete: (task: Task) => void;
  onOpen: (task: Task) => void;
}

/** Một dòng task trong agenda: dải màu + chip project để luôn biết task thuộc project nào. */
export function AgendaRow({
  task,
  project,
  parentTitle,
  mode,
  today0,
  completing,
  completeBlockedReason,
  onComplete,
  onOpen,
}: AgendaRowProps) {
  const { t, i18n } = useTranslation();
  const color = projectColor(project);
  const due = task.dueAt ? new Date(task.dueAt) : null;

  let timeLabel: string | null = null;
  if (due && mode === 'overdue') {
    timeLabel = t('agendaPage.lateDays', { count: Math.max(1, calendarDaysBetween(due, today0)) });
  } else if (due && !isLocalMidnight(due)) {
    timeLabel = due.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' });
  }

  return (
    <Box
      className={classes.row}
      data-completing={completing || undefined}
      style={{ borderLeftColor: color }}
      onClick={() => onOpen(task)}
    >
      <Tooltip label={completeBlockedReason ?? t('agendaPage.markDone')} withArrow multiline maw={260}>
        <ActionIcon
          variant="subtle"
          color={completing ? 'teal' : 'gray'}
          radius="xl"
          aria-label={t('agendaPage.markDone')}
          disabled={completeBlockedReason !== null || completing}
          onClick={(event) => {
            event.stopPropagation();
            onComplete(task);
          }}
        >
          {completing ? <IconCircleCheck size={19} /> : <IconCircle size={19} />}
        </ActionIcon>
      </Tooltip>

      <Box className={classes.body}>
        <Box className={classes.titleLine}>
          <Text fz={13.5} fw={600} c={tokens.text} className={classes.title} td={completing ? 'line-through' : undefined}>
            {task.title}
          </Text>
          <Box className={classes.meta}>
            <Tooltip label={project?.name ?? t('agendaPage.unknownProject')} withArrow openDelay={300}>
              <Box component="span" className={classes.projectChip}>
                <span className={classes.dot} style={{ background: color }} />
                <Text component="span" fz={11} fw={600} className={classes.projectName}>
                  {project?.name ?? t('agendaPage.unknownProject')}
                </Text>
              </Box>
            </Tooltip>
            {task.priority !== 'TASK_PRIORITY_MEDIUM' && task.priority !== 'TASK_PRIORITY_UNSPECIFIED' ? (
              <PriorityBadge priority={task.priority} />
            ) : null}
          </Box>
        </Box>
        {parentTitle ? (
          <Group gap={4} wrap="nowrap" mt={2}>
            <IconCornerDownRight size={12} style={{ color: tokens.textFaint, flexShrink: 0 }} />
            <Text fz={11.5} c={tokens.textMuted} truncate>
              {t('agendaPage.inParent', { title: parentTitle })}
            </Text>
          </Group>
        ) : null}
      </Box>

      {timeLabel ? (
        <Text fz={12} fw={mode === 'overdue' ? 700 : 500} c={mode === 'overdue' ? tokens.danger : tokens.textMuted} className={classes.time}>
          {timeLabel}
        </Text>
      ) : null}
    </Box>
  );
}

export default AgendaRow;
