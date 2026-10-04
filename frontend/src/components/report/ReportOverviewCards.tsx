import { SimpleGrid, Text, ThemeIcon } from '@mantine/core';
import {
  IconAlertTriangle,
  IconCalendarStats,
  IconChecklist,
  IconCircleCheck,
  IconClockHour4,
  IconFilePlus,
  IconProgress,
  IconStack2,
} from '@tabler/icons-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { formatPercent } from '../../lib/format';
import { tokens } from '../../theme';
import type { ReportOverview } from '../../types';
import classes from './report.module.css';

interface StatCard {
  key: string;
  label: string;
  value: number;
  hint?: string;
  color: string;
  icon: ReactNode;
}

export function ReportOverviewCards({ overview }: { overview: ReportOverview }) {
  const { t } = useTranslation();

  const cards: StatCard[] = [
    {
      key: 'total',
      label: t('reportPage.statTotal'),
      value: overview.totalTasks,
      hint: t('reportPage.statRootHint', {
        roots: overview.rootTasks,
        subtasks: overview.subtaskCount,
      }),
      color: 'brand',
      icon: <IconStack2 size={16} />,
    },
    {
      key: 'done',
      label: t('reportPage.statDone'),
      value: overview.doneTasks,
      hint: t('reportPage.statDoneHint', { rate: formatPercent(overview.completionRate) }),
      color: 'teal',
      icon: <IconCircleCheck size={16} />,
    },
    {
      key: 'inProgress',
      label: t('reportPage.statInProgress'),
      value: overview.inProgressTasks,
      color: 'blue',
      icon: <IconProgress size={16} />,
    },
    {
      key: 'todo',
      label: t('reportPage.statTodo'),
      value: overview.todoTasks,
      color: 'grape',
      icon: <IconChecklist size={16} />,
    },
    {
      key: 'overdue',
      label: t('reportPage.statOverdue'),
      value: overview.overdueTasks,
      hint: t('reportPage.statOverdueHint', { rate: formatPercent(overview.overdueRate) }),
      color: 'red',
      icon: <IconAlertTriangle size={16} />,
    },
    {
      key: 'dueSoon',
      label: t('reportPage.statDueSoon'),
      value: overview.dueSoonTasks,
      color: 'orange',
      icon: <IconClockHour4 size={16} />,
    },
    {
      key: 'created',
      label: t('reportPage.statCreated'),
      value: overview.createdInRange,
      color: 'indigo',
      icon: <IconFilePlus size={16} />,
    },
    {
      key: 'completed',
      label: t('reportPage.statCompleted'),
      value: overview.completedInRange,
      color: 'green',
      icon: <IconCalendarStats size={16} />,
    },
  ];

  return (
    <SimpleGrid cols={{ base: 2, md: 4 }} spacing="sm">
      {cards.map((card) => (
        <div key={card.key} className={classes.statCard}>
          <ThemeIcon size={28} radius={9} variant="light" color={card.color}>
            {card.icon}
          </ThemeIcon>
          <Text className={classes.statLabel} mt={9}>
            {card.label}
          </Text>
          <Text className={classes.statValue}>{card.value}</Text>
          {card.hint ? (
            <Text className={classes.statHint} c={tokens.textFaint}>
              {card.hint}
            </Text>
          ) : null}
        </div>
      ))}
    </SimpleGrid>
  );
}
