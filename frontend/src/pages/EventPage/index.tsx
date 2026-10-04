import { useEffect, useMemo, useState } from 'react';
import {
  ActionIcon,
  Badge,
  Box,
  Button,
  Flex,
  Group,
  SegmentedControl,
  Table,
  Text,
  ThemeIcon,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconCalendarEvent, IconChevronRight, IconPencil, IconPlus } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { getEvent } from '../../api/event';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import EventFormModal from '../../components/event/EventFormModal';
import { useEvent, useNoticeFocus, useSession } from '../../context';
import { EVENT_STATUS_COLOR, EVENT_STATUS_LABEL_KEY } from '../../lib/tokens';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import { EVENT_STATUSES, type Event, type EventStatus } from '../../types';
import classes from './EventPage.module.css';

type StatusFilter = 'all' | EventStatus;

export function EventPage() {
  const { t, i18n } = useTranslation();
  const session = useSession();
  const eventState = useEvent();

  const [modalOpened, setModalOpened] = useState(false);
  const [editing, setEditing] = useState<Event | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  const { profileId, refresh } = eventState;

  useEffect(() => {
    if (profileId) {
      refresh();
    }
  }, [profileId, refresh]);

  const locale = i18n.resolvedLanguage?.startsWith('vi') ? 'vi-VN' : 'en-GB';

  const events = useMemo(() => {
    const filtered =
      statusFilter === 'all'
        ? eventState.events
        : eventState.events.filter((item) => item.status === statusFilter);
    return [...filtered].sort((a, b) => a.startAt.localeCompare(b.startAt));
  }, [eventState.events, statusFilter]);

  const loading = session.loading || eventState.loading;
  const error = session.error ?? eventState.error;

  const refreshAll = () => {
    session.refresh();
    eventState.refresh();
  };

  const openCreate = () => {
    setEditing(null);
    setModalOpened(true);
  };

  const openEdit = (item: Event) => {
    setEditing(item);
    setModalOpened(true);
  };

  const closeModal = () => {
    setModalOpened(false);
    setEditing(null);
  };

  useNoticeFocus<Event>(
    async (id) => eventState.events.find((item) => item.id === id) ?? (await getEvent(id).catch(() => null)),
    (item) => openEdit(item),
  );

  const handleDelete = async (item: Event) => {
    setDeleting(true);
    try {
      await eventState.remove(item.id);
      notifications.show({
        title: t('eventPage.deleted'),
        message: item.title,
        color: 'teal',
      });
      closeModal();
    } catch (cause) {
      notifications.show({
        title: t('eventPage.deleteFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setDeleting(false);
    }
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

  const formatRange = (item: Event): string => {
    const start = new Date(item.startAt);
    if (Number.isNaN(start.getTime())) {
      return '—';
    }
    const date = start.toLocaleDateString(locale, {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
    if (item.allDay) {
      return `${date} · ${t('eventPage.allDay')}`;
    }
    const startTime = start.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' });
    if (!item.endAt) {
      return `${date} ${startTime}`;
    }
    const end = new Date(item.endAt);
    if (Number.isNaN(end.getTime())) {
      return `${date} ${startTime}`;
    }
    const endTime = end.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' });
    return `${date} ${startTime} – ${endTime}`;
  };

  const ready = !error && session.profile !== null;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" pb="sm" className={classes.headerBar}>
        <Group gap={5} wrap="nowrap">
          <Text fz={12} c={tokens.textMuted}>
            {t('eventPage.breadcrumb')}
          </Text>
          <IconChevronRight size={13} style={{ color: tokens.textFaint }} />
          <Text fz={12} fw={600} c={tokens.text}>
            {t('eventPage.title')}
          </Text>
        </Group>

        <Group justify="space-between" align="center" mt={10} wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconCalendarEvent size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('eventPage.title')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('eventPage.subtitle')}
              </Text>
            </div>
          </Group>

          <Group gap="sm">
            <SegmentedControl
              size="xs"
              value={statusFilter}
              onChange={(value) => setStatusFilter(value as StatusFilter)}
              data={[
                { value: 'all', label: t('eventPage.filterAll') },
                ...EVENT_STATUSES.map((value) => ({
                  value,
                  label: t(EVENT_STATUS_LABEL_KEY[value]),
                })),
              ]}
            />
            <Button
              leftSection={<IconPlus size={15} />}
              disabled={!session.profile}
              onClick={openCreate}
            >
              {t('eventPage.newEvent')}
            </Button>
          </Group>
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        {error ? <ApiErrorAlert message={error} onRetry={refreshAll} /> : null}
        {!error && loading && eventState.events.length === 0 ? <LoadingBlock /> : null}

        {!error && !loading && !session.profile ? (
          <CenteredPanel
            icon={IconCalendarEvent}
            title={t('boardPage.noProfileTitle')}
            description={t('boardPage.noProfileDesc')}
            action={
              <Button leftSection={<IconPlus size={15} />} onClick={handleCreateProfile}>
                {t('boardPage.createProfile')}
              </Button>
            }
          />
        ) : null}

        {ready && events.length === 0 ? (
          <CenteredPanel
            icon={IconCalendarEvent}
            title={t('eventPage.emptyTitle')}
            description={t('eventPage.emptyDesc')}
            action={
              <Button leftSection={<IconPlus size={15} />} onClick={openCreate}>
                {t('eventPage.newEvent')}
              </Button>
            }
          />
        ) : null}

        {ready && events.length > 0 ? (
          <Box className={classes.panel}>
            <Box px="md" py="sm" className={classes.panelHeader}>
              <Text fz={13.5} fw={700} c={tokens.text}>
                {t('eventPage.panelTitle', { n: events.length })}
              </Text>
            </Box>
            <Table highlightOnHover verticalSpacing="sm" horizontalSpacing="md">
              <Table.Thead>
                <Table.Tr className={classes.headRow}>
                  <Table.Th>{t('eventPage.colEvent')}</Table.Th>
                  <Table.Th w={230}>{t('eventPage.colTime')}</Table.Th>
                  <Table.Th w={190}>{t('eventPage.colLocation')}</Table.Th>
                  <Table.Th w={140}>{t('eventPage.colStatus')}</Table.Th>
                  <Table.Th w={70} />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {events.map((item) => (
                  <Table.Tr key={item.id}>
                    <Table.Td className={classes.titleCell} onClick={() => openEdit(item)}>
                      <Text fz={13} fw={600} c={tokens.text}>
                        {item.title}
                      </Text>
                      {item.description ? (
                        <Text fz={12} c={tokens.textMuted} lineClamp={1}>
                          {item.description}
                        </Text>
                      ) : null}
                    </Table.Td>
                    <Table.Td>
                      <Text fz={12.5} c={tokens.textMuted}>
                        {formatRange(item)}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text fz={12.5} c={tokens.textMuted}>
                        {item.location || '—'}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge size="sm" variant="light" color={EVENT_STATUS_COLOR[item.status]}>
                        {t(EVENT_STATUS_LABEL_KEY[item.status])}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <ActionIcon variant="subtle" color="gray" onClick={() => openEdit(item)}>
                        <IconPencil size={15} />
                      </ActionIcon>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Box>
        ) : null}
      </Box>

      <EventFormModal
        opened={modalOpened}
        onClose={closeModal}
        event={editing}
        onDelete={handleDelete}
        deleting={deleting}
      />
    </Flex>
  );
}

export default EventPage;
