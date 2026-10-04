import { useEffect, useState } from 'react';
import {
  ActionIcon,
  Box,
  Button,
  Checkbox,
  Flex,
  Group,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  ThemeIcon,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import {
  IconChartHistogram,
  IconChevronRight,
  IconPlus,
  IconRefresh,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { ActivityStats } from '../../components/report/ActivityStats';
import { ReportOverviewCards } from '../../components/report/ReportOverviewCards';
import { StatusBreakdownChart } from '../../components/report/StatusBreakdownChart';
import { TaskTrendChart } from '../../components/report/TaskTrendChart';
import reportClasses from '../../components/report/report.module.css';
import { useReport, useSession } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import { REPORT_INTERVALS, type ReportInterval } from '../../types';
import classes from './ReportPage.module.css';

const INTERVAL_LABEL_KEY: Record<ReportInterval, string> = {
  REPORT_INTERVAL_UNSPECIFIED: 'reportPage.intervalDay',
  REPORT_INTERVAL_DAY: 'reportPage.intervalDay',
  REPORT_INTERVAL_WEEK: 'reportPage.intervalWeek',
  REPORT_INTERVAL_MONTH: 'reportPage.intervalMonth',
};

function isoDate(date: Date): string {
  return date.toISOString().slice(0, 10);
}

function daysAgo(days: number): Date {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() - days);
  return date;
}

export function ReportPage() {
  const { t } = useTranslation();
  const session = useSession();
  const report = useReport();

  const [preset, setPreset] = useState('30');
  const [from, setFrom] = useState(() => isoDate(daysAgo(29)));
  const [to, setTo] = useState(() => isoDate(new Date()));
  const [interval, setInterval] = useState<ReportInterval>('REPORT_INTERVAL_DAY');
  const [includeArchived, setIncludeArchived] = useState(false);

  const { profileId, applyRange, refresh } = report;

  useEffect(() => {
    if (!profileId) {
      return;
    }
    applyRange({
      from: `${from}T00:00:00Z`,
      to: `${to}T23:59:59Z`,
      interval,
      includeArchived,
    });
    // applyRange đổi identity khi store.range đổi; chỉ chạy lại theo filter người dùng.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileId, from, to, interval, includeArchived]);

  const applyPreset = (days: string) => {
    setPreset(days);
    const count = Number(days);
    setFrom(isoDate(daysAgo(count - 1)));
    setTo(isoDate(new Date()));
  };

  const handleCreateProfile = async () => {
    try {
      await session.createProfile({
        email: 'me@example.com',
        displayName: t('boardPage.defaultProfileName'),
      });
    } catch (cause) {
      notifications.show({
        title: t('boardPage.createProfileFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    }
  };

  const error = session.error ?? report.error;
  const loading = session.loading || (report.loading && report.overview === null);
  const ready = !error && session.profile !== null && report.overview !== null;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" pb="sm" className={classes.headerBar}>
        <Group gap={5} wrap="nowrap">
          <Text fz={12} c={tokens.textMuted}>
            {t('nav.groups.dashboard')}
          </Text>
          <IconChevronRight size={13} style={{ color: tokens.textFaint }} />
          <Text fz={12} fw={600} c={tokens.text}>
            {t('reportPage.title')}
          </Text>
        </Group>

        <Group justify="space-between" align="flex-end" mt={10} wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconChartHistogram size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('reportPage.title')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('reportPage.subtitle')}
              </Text>
            </div>
          </Group>

          <Group gap="sm" align="flex-end" wrap="wrap">
            <div>
              <div className={classes.filterLabel}>{t('reportPage.rangeLabel')}</div>
              <SegmentedControl
                size="xs"
                value={preset}
                onChange={applyPreset}
                data={[
                  { value: '7', label: t('reportPage.range7') },
                  { value: '30', label: t('reportPage.range30') },
                  { value: '90', label: t('reportPage.range90') },
                ]}
              />
            </div>

            <div>
              <div className={classes.filterLabel}>{t('reportPage.fromLabel')}</div>
              <TextInput
                size="xs"
                type="date"
                className={classes.dateInput}
                value={from}
                max={to}
                onChange={(event) => {
                  setPreset('custom');
                  setFrom(event.currentTarget.value);
                }}
              />
            </div>

            <div>
              <div className={classes.filterLabel}>{t('reportPage.toLabel')}</div>
              <TextInput
                size="xs"
                type="date"
                className={classes.dateInput}
                value={to}
                min={from}
                onChange={(event) => {
                  setPreset('custom');
                  setTo(event.currentTarget.value);
                }}
              />
            </div>

            <div>
              <div className={classes.filterLabel}>{t('reportPage.intervalLabel')}</div>
              <SegmentedControl
                size="xs"
                value={interval}
                onChange={(value) => setInterval(value as ReportInterval)}
                data={REPORT_INTERVALS.map((value) => ({
                  value,
                  label: t(INTERVAL_LABEL_KEY[value]),
                }))}
              />
            </div>

            <Checkbox
              size="xs"
              checked={includeArchived}
              onChange={(event) => setIncludeArchived(event.currentTarget.checked)}
              label={t('reportPage.includeArchived')}
            />

            <ActionIcon
              variant="light"
              color="brand"
              size="lg"
              aria-label={t('boardHeader.refresh')}
              onClick={refresh}
            >
              <IconRefresh size={16} />
            </ActionIcon>
          </Group>
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        {error ? <ApiErrorAlert message={error} onRetry={refresh} /> : null}
        {!error && loading ? <LoadingBlock /> : null}

        {!error && !loading && session.profile === null ? (
          <CenteredPanel
            icon={IconChartHistogram}
            title={t('boardPage.noProfileTitle')}
            description={t('boardPage.noProfileDesc')}
            action={
              <Button leftSection={<IconPlus size={15} />} onClick={handleCreateProfile}>
                {t('boardPage.createProfile')}
              </Button>
            }
          />
        ) : null}

        {ready ? (
          <Stack gap="md">
            <ReportOverviewCards overview={report.overview!} />

            <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="md" className={classes.grid}>
              <div className={reportClasses.panel}>
                <div className={reportClasses.panelHeader}>
                  <div>
                    <div className={reportClasses.panelTitle}>
                      {t('reportPage.statusChartTitle')}
                    </div>
                    <div className={reportClasses.panelDesc}>
                      {t('reportPage.statusChartDesc', { n: report.breakdown.total })}
                    </div>
                  </div>
                </div>
                <div className={`${reportClasses.panelBody} ${classes.panelBodyDonut}`}>
                  <StatusBreakdownChart breakdown={report.breakdown} />
                </div>
              </div>

              <div className={reportClasses.panel}>
                <div className={reportClasses.panelHeader}>
                  <div>
                    <div className={reportClasses.panelTitle}>
                      {t('reportPage.trendChartTitle')}
                    </div>
                    <div className={reportClasses.panelDesc}>
                      {t('reportPage.trendChartDesc')}
                    </div>
                  </div>
                </div>
                <div className={reportClasses.panelBody}>
                  {report.series ? <TaskTrendChart series={report.series} /> : null}
                </div>
              </div>
            </SimpleGrid>

            {report.activity ? <ActivityStats activity={report.activity} /> : null}
          </Stack>
        ) : null}
      </Box>
    </Flex>
  );
}

export default ReportPage;
