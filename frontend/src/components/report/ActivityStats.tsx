import { SimpleGrid, Text, ThemeIcon } from '@mantine/core';
import {
  IconArchive,
  IconCalendarEvent,
  IconCalendarMonth,
  IconCalendarStats,
  IconClockHour4,
  IconFilePlus,
  IconProgress,
} from '@tabler/icons-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { hasText } from '../../lib/format';
import { BACKLOG_CATEGORY_LABEL_KEY } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { ReportActivity } from '../../types';
import { DonutBreakdown, type DonutSlice } from './DonutBreakdown';
import classes from './report.module.css';

const EVENT_COLORS = {
  planned: '#4c6ef5',
  confirmed: '#12b886',
  cancelled: '#e5484d',
};

const BACKLOG_COLORS = {
  new: '#4353e8',
  triaged: '#12b886',
  archived: '#9aa0ae',
};

interface StatCard {
  key: string;
  label: string;
  value: number;
  hint?: string;
  color: string;
  icon: ReactNode;
}

function percent(value: number, total: number): number {
  if (total <= 0) {
    return 0;
  }
  return Math.round((value / total) * 10000) / 100;
}

export function ActivityStats({ activity }: { activity: ReportActivity }) {
  const { t } = useTranslation();

  const eventTotal =
    activity.events.plannedEvents +
    activity.events.confirmedEvents +
    activity.events.cancelledEvents;

  const eventSlices: DonutSlice[] = [
    {
      id: 'planned',
      label: t('eventStatus.planned'),
      value: activity.events.plannedEvents,
      color: EVENT_COLORS.planned,
      percentage: percent(activity.events.plannedEvents, eventTotal),
    },
    {
      id: 'confirmed',
      label: t('eventStatus.confirmed'),
      value: activity.events.confirmedEvents,
      color: EVENT_COLORS.confirmed,
      percentage: percent(activity.events.confirmedEvents, eventTotal),
    },
    {
      id: 'cancelled',
      label: t('eventStatus.cancelled'),
      value: activity.events.cancelledEvents,
      color: EVENT_COLORS.cancelled,
      percentage: percent(activity.events.cancelledEvents, eventTotal),
    },
  ];

  const backlogTotal =
    activity.backlogs.newBacklogs +
    activity.backlogs.triagedBacklogs +
    activity.backlogs.archivedBacklogs;

  const backlogSlices: DonutSlice[] = [
    {
      id: 'new',
      label: t('backlogStatus.new'),
      value: activity.backlogs.newBacklogs,
      color: BACKLOG_COLORS.new,
      percentage: percent(activity.backlogs.newBacklogs, backlogTotal),
    },
    {
      id: 'triaged',
      label: t('backlogStatus.triaged'),
      value: activity.backlogs.triagedBacklogs,
      color: BACKLOG_COLORS.triaged,
      percentage: percent(activity.backlogs.triagedBacklogs, backlogTotal),
    },
    {
      id: 'archived',
      label: t('backlogStatus.archived'),
      value: activity.backlogs.archivedBacklogs,
      color: BACKLOG_COLORS.archived,
      percentage: percent(activity.backlogs.archivedBacklogs, backlogTotal),
    },
  ];

  const categories = activity.backlogs.byCategory;
  const maxCategory = categories.reduce((max, item) => Math.max(max, item.count), 0);

  const cards: StatCard[] = [
    {
      key: 'eventsTotal',
      label: t('reportPage.statEvents'),
      value: activity.events.totalEvents,
      hint: t('reportPage.statEventsHint', {
        planned: activity.events.plannedEvents,
        confirmed: activity.events.confirmedEvents,
      }),
      color: 'brand',
      icon: <IconCalendarEvent size={16} />,
    },
    {
      key: 'eventsUpcoming',
      label: t('reportPage.statEventsUpcoming'),
      value: activity.events.upcomingEvents,
      color: 'blue',
      icon: <IconClockHour4 size={16} />,
    },
    {
      key: 'eventsAllDay',
      label: t('reportPage.statEventsAllDay'),
      value: activity.events.allDayEvents,
      color: 'grape',
      icon: <IconCalendarStats size={16} />,
    },
    {
      key: 'schedulesTotal',
      label: t('reportPage.statSchedules'),
      value: activity.schedules.totalSchedules,
      hint: t('reportPage.statSchedulesHint', {
        allDay: activity.schedules.allDaySchedules,
        upcoming: activity.schedules.upcomingSchedules,
      }),
      color: 'indigo',
      icon: <IconCalendarMonth size={16} />,
    },
    {
      key: 'schedulesToday',
      label: t('reportPage.statSchedulesToday'),
      value: activity.schedules.todaySchedules,
      color: 'cyan',
      icon: <IconClockHour4 size={16} />,
    },
    {
      key: 'backlogsTotal',
      label: t('reportPage.statBacklogs'),
      value: activity.backlogs.totalBacklogs,
      color: 'orange',
      icon: <IconArchive size={16} />,
    },
    {
      key: 'backlogsNew',
      label: t('reportPage.statBacklogsNew'),
      value: activity.backlogs.newBacklogs,
      color: 'brand',
      icon: <IconFilePlus size={16} />,
    },
    {
      key: 'backlogsTriaged',
      label: t('reportPage.statBacklogsTriaged'),
      value: activity.backlogs.triagedBacklogs,
      color: 'teal',
      icon: <IconProgress size={16} />,
    },
  ];

  return (
    <div>
      <Text className={classes.sectionTitle}>{t('reportPage.activityTitle')}</Text>
      <Text className={classes.sectionDesc}>{t('reportPage.activityDesc')}</Text>

      <SimpleGrid cols={{ base: 2, md: 4 }} spacing="sm" mt="sm">
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

      <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="md" mt="md">
        <div className={classes.panel}>
          <div className={classes.panelHeader}>
            <div>
              <div className={classes.panelTitle}>{t('reportPage.eventChartTitle')}</div>
              <div className={classes.panelDesc}>{t('reportPage.eventChartDesc')}</div>
            </div>
          </div>
          <div className={classes.panelBody}>
            <DonutBreakdown
              slices={eventSlices}
              total={eventTotal}
              centerLabel={t('reportPage.donutEvents')}
              emptyLabel={t('reportPage.noActivityData')}
              ariaLabel={t('reportPage.eventChartTitle')}
            />
          </div>
        </div>

        <div className={classes.panel}>
          <div className={classes.panelHeader}>
            <div>
              <div className={classes.panelTitle}>{t('reportPage.backlogChartTitle')}</div>
              <div className={classes.panelDesc}>{t('reportPage.backlogChartDesc')}</div>
            </div>
          </div>
          <div className={classes.panelBody}>
            <DonutBreakdown
              slices={backlogSlices}
              total={backlogTotal}
              centerLabel={t('reportPage.donutBacklogs')}
              emptyLabel={t('reportPage.noActivityData')}
              ariaLabel={t('reportPage.backlogChartTitle')}
            />
          </div>
        </div>

        <div className={classes.panel}>
          <div className={classes.panelHeader}>
            <div>
              <div className={classes.panelTitle}>{t('reportPage.backlogCategoryTitle')}</div>
              <div className={classes.panelDesc}>{t('reportPage.backlogCategoryDesc')}</div>
            </div>
          </div>
          <div className={classes.panelBody}>
            {categories.length === 0 ? (
              <Text className={classes.empty}>{t('reportPage.noActivityData')}</Text>
            ) : (
              <div className={classes.categoryList}>
                {categories.map((item) => {
                  const labelKey = hasText(item.category)
                    ? BACKLOG_CATEGORY_LABEL_KEY[item.category]
                    : undefined;
                  const width = maxCategory > 0 ? (item.count / maxCategory) * 100 : 0;
                  return (
                    <div key={item.category} className={classes.categoryRow}>
                      <div className={classes.categoryHead}>
                        <span className={classes.categoryName}>
                          {labelKey ? t(labelKey) : item.category}
                        </span>
                        <span className={classes.categoryValue}>{item.count}</span>
                      </div>
                      <div className={classes.barTrack}>
                        <div className={classes.barFill} style={{ width: `${width}%` }} />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </SimpleGrid>
    </div>
  );
}
