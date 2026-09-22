import { useEffect, useState } from 'react';
import { Badge, Box, Button, Flex, Group, Stack, Table, Text, ThemeIcon } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import {
  IconArrowsExchange,
  IconArrowRight,
  IconChevronRight,
  IconSparkles,
} from '@tabler/icons-react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useAppDispatch, useSession, useStatuses } from '../../context';
import { fetchStatusesPage } from '../../context/statuses/statusesSlice';
import { CATEGORY_LABEL_KEY, statusColor } from '../../lib/tokens';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import classes from './StatusesPage.module.css';

function Panel({
  title,
  description,
  children,
}: {
  title: string;
  description?: ReactNode;
  children: ReactNode;
}) {
  return (
    <Box className={classes.panel}>
      <Box px="md" py="sm" className={classes.panelHeader}>
        <Text fz={13.5} fw={700} c={tokens.text}>
          {title}
        </Text>
        {description ? (
          <Text fz={12} c={tokens.textMuted} mt={2}>
            {description}
          </Text>
        ) : null}
      </Box>
      {children}
    </Box>
  );
}

export function StatusesPage() {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const {
    profileId,
    loading: sessionLoading,
    error: sessionError,
    refresh: refreshSession,
  } = useSession();
  const statusesState = useStatuses();
  const { statuses, transitions, statusById, tasks, seedDefaultStatuses, workspaceId } = statusesState;
  const [busy, setBusy] = useState(false);

  const loading = sessionLoading || statusesState.loading;
  const error = sessionError ?? statusesState.error;
  const refresh = () => {
    refreshSession();
    statusesState.refresh();
  };

  useEffect(() => {
    if (profileId) {
      void dispatch(fetchStatusesPage({ profileId, workspaceId }));
    }
  }, [profileId, workspaceId, dispatch]);

  const handleSeed = async () => {
    setBusy(true);
    try {
      await seedDefaultStatuses();
      notifications.show({
        title: t('statusesPage.seedDoneTitle'),
        message: t('statusesPage.seedDoneMessage'),
        color: 'teal',
      });
    } catch (cause) {
      notifications.show({
        title: t('statusesPage.seedFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setBusy(false);
    }
  };

  const usageCount = (statusId: string) => tasks.filter((task) => task.statusId === statusId).length;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" className={classes.headerBar}>
        <Group gap={5} wrap="nowrap">
          <Text fz={12} c={tokens.textMuted}>
            {t('statusesPage.breadcrumb')}
          </Text>
          <IconChevronRight size={13} style={{ color: tokens.textFaint }} />
          <Text fz={12} fw={600} c={tokens.text}>
            {t('statusesPage.title')}
          </Text>
        </Group>

        <Group justify="space-between" align="center" mt={10} mb="md" wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconArrowsExchange size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('statusesPage.title')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('statusesPage.subtitle')}
              </Text>
            </div>
          </Group>

          {statuses.length === 0 && !loading && !error ? (
            <Button loading={busy} leftSection={<IconSparkles size={15} />} onClick={handleSeed}>
              {t('statusesPage.seedStatuses')}
            </Button>
          ) : null}
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        {error ? <ApiErrorAlert message={error} onRetry={refresh} /> : null}
        {!error && loading && statuses.length === 0 ? <LoadingBlock /> : null}

        {!error && !loading && statuses.length === 0 ? (
          <CenteredPanel
            icon={IconSparkles}
            title={t('statusesPage.emptyTitle')}
            description={t('statusesPage.emptyDesc')}
            action={
              <Button loading={busy} leftSection={<IconSparkles size={15} />} onClick={handleSeed}>
                {t('statusesPage.seedStatuses')}
              </Button>
            }
          />
        ) : null}

        {statuses.length > 0 ? (
          <Stack gap="lg">
            <Panel
              title={t('statusesPage.statusesPanel', { n: statuses.length })}
              description={t('statusesPage.statusesPanelDesc')}
            >
              <Table highlightOnHover verticalSpacing="sm" horizontalSpacing="md">
                <Table.Thead>
                  <Table.Tr className={classes.headRow}>
                    <Table.Th>{t('statusesPage.colStatus')}</Table.Th>
                    <Table.Th w={140}>{t('statusesPage.colSlug')}</Table.Th>
                    <Table.Th w={170}>{t('statusesPage.colCategory')}</Table.Th>
                    <Table.Th w={110}>{t('statusesPage.colDefault')}</Table.Th>
                    <Table.Th w={110}>{t('statusesPage.colTerminal')}</Table.Th>
                    <Table.Th w={90}>{t('statusesPage.colTasks')}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {statuses.map((status, index) => (
                    <Table.Tr key={status.id}>
                      <Table.Td>
                        <Group gap={8} wrap="nowrap">
                          <Box
                            className={classes.dot}
                            style={{ background: statusColor(status.color, index) }}
                          />
                          <Text fz={13} fw={600} c={tokens.text}>
                            {status.name}
                          </Text>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Text fz={12.5} c={tokens.textMuted} ff="monospace">
                          {status.slug}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Text fz={12.5} c={tokens.textMuted}>
                          {t(CATEGORY_LABEL_KEY[status.category])}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        {status.isDefault ? (
                          <Badge size="sm" variant="light" color="brand">
                            {t('statusesPage.badgeDefault')}
                          </Badge>
                        ) : (
                          <Text fz={12.5} c={tokens.textFaint}>
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        {status.isTerminal ? (
                          <Badge size="sm" variant="light" color="grape">
                            {t('statusesPage.badgeTerminal')}
                          </Badge>
                        ) : (
                          <Text fz={12.5} c={tokens.textFaint}>
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Text fz={12.5} fw={600} c={tokens.textMuted}>
                          {usageCount(status.id)}
                        </Text>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </Panel>

            <Panel
              title={t('statusesPage.rulesPanel', { n: transitions.length })}
              description={t('statusesPage.rulesPanelDesc')}
            >
              {transitions.length === 0 ? (
                <Text fz={12.5} c={tokens.textMuted} p="md">
                  {t('statusesPage.rulesEmpty')}
                </Text>
              ) : (
                <Table highlightOnHover verticalSpacing="sm" horizontalSpacing="md">
                  <Table.Thead>
                    <Table.Tr className={classes.headRow}>
                      <Table.Th>{t('statusesPage.colFrom')}</Table.Th>
                      <Table.Th w={60} />
                      <Table.Th>{t('statusesPage.colTo')}</Table.Th>
                      <Table.Th w={120}>{t('statusesPage.colActive')}</Table.Th>
                      <Table.Th w={140}>{t('statusesPage.colRequiresNote')}</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {transitions.map((transition) => {
                      const from = statusById.get(transition.fromStatusId);
                      const to = statusById.get(transition.toStatusId);
                      const fromIndex = statuses.findIndex(
                        (item) => item.id === transition.fromStatusId,
                      );
                      const toIndex = statuses.findIndex((item) => item.id === transition.toStatusId);

                      return (
                        <Table.Tr key={transition.id}>
                          <Table.Td>
                            <Group gap={8} wrap="nowrap">
                              <Box
                                className={classes.dot}
                                style={{
                                  background: statusColor(from?.color, fromIndex < 0 ? 0 : fromIndex),
                                }}
                              />
                              <Text fz={13} fw={600} c={tokens.text}>
                                {from?.name ?? '—'}
                              </Text>
                            </Group>
                          </Table.Td>
                          <Table.Td>
                            <IconArrowRight size={15} style={{ color: tokens.textFaint }} />
                          </Table.Td>
                          <Table.Td>
                            <Group gap={8} wrap="nowrap">
                              <Box
                                className={classes.dot}
                                style={{
                                  background: statusColor(to?.color, toIndex < 0 ? 0 : toIndex),
                                }}
                              />
                              <Text fz={13} fw={600} c={tokens.text}>
                                {to?.name ?? '—'}
                              </Text>
                            </Group>
                          </Table.Td>
                          <Table.Td>
                            <Badge
                              size="sm"
                              variant="light"
                              color={transition.isActive ? 'teal' : 'gray'}
                            >
                              {transition.isActive
                                ? t('statusesPage.badgeActive')
                                : t('statusesPage.badgeOff')}
                            </Badge>
                          </Table.Td>
                          <Table.Td>
                            <Text fz={12.5} c={tokens.textMuted}>
                              {transition.requiresNote ? t('statusesPage.yes') : '—'}
                            </Text>
                          </Table.Td>
                        </Table.Tr>
                      );
                    })}
                  </Table.Tbody>
                </Table>
              )}
            </Panel>
          </Stack>
        ) : null}
      </Box>
    </Flex>
  );
}

export default StatusesPage;
