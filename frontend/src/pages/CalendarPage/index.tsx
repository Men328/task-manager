import { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Button, Flex, Group, Text, ThemeIcon } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconCalendarMonth, IconChevronRight, IconPlus } from '@tabler/icons-react';
import type { DateClickArg } from '@fullcalendar/interaction';
import interactionPlugin from '@fullcalendar/interaction';
import type { DatesSetArg, EventChangeArg, EventClickArg, EventInput } from '@fullcalendar/core';
import viLocale from '@fullcalendar/core/locales/vi';
import dayGridPlugin from '@fullcalendar/daygrid';
import FullCalendar from '@fullcalendar/react';
import timeGridPlugin from '@fullcalendar/timegrid';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { getSchedule } from '../../api/calendar';
import ScheduleFormModal from '../../components/calendar/ScheduleFormModal';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useCalendar, useNoticeFocus, useSession } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { Schedule } from '../../types';
import classes from './CalendarPage.module.css';

export function CalendarPage() {
  const { t, i18n } = useTranslation();
  const session = useSession();
  const calendar = useCalendar();

  const [modalOpened, setModalOpened] = useState(false);
  const [editing, setEditing] = useState<Schedule | null>(null);
  const [defaultStart, setDefaultStart] = useState<string | null>(null);
  const [defaultAllDay, setDefaultAllDay] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const rangeRef = useRef<{ from: string; to: string } | null>(null);
  const loadedProfileRef = useRef<string | null>(null);
  const { profileId, loadRange } = calendar;

  useEffect(() => {
    if (profileId && rangeRef.current && loadedProfileRef.current !== profileId) {
      loadedProfileRef.current = profileId;
      loadRange(rangeRef.current.from, rangeRef.current.to);
    }
  }, [profileId, loadRange]);

  const events = useMemo<EventInput[]>(
    () =>
      calendar.schedules.map((schedule) => ({
        id: schedule.id,
        title: schedule.title,
        start: schedule.startAt,
        end: schedule.endAt ?? undefined,
        allDay: schedule.allDay,
        backgroundColor: schedule.color || undefined,
        borderColor: schedule.color || undefined,
        extendedProps: { description: schedule.description, location: schedule.location },
      })),
    [calendar.schedules],
  );

  const loading = session.loading || calendar.loading;
  const error = session.error ?? calendar.error;

  const refresh = () => {
    session.refresh();
    calendar.refresh();
  };

  const openCreate = (start: string | null, allDay: boolean) => {
    setEditing(null);
    setDefaultStart(start);
    setDefaultAllDay(allDay);
    setModalOpened(true);
  };

  const closeModal = () => {
    setModalOpened(false);
    setEditing(null);
  };

  const handleDatesSet = (arg: DatesSetArg) => {
    rangeRef.current = { from: arg.startStr, to: arg.endStr };
    if (profileId) {
      loadedProfileRef.current = profileId;
      loadRange(arg.startStr, arg.endStr);
    }
  };

  const handleDateClick = (arg: DateClickArg) => {
    openCreate(arg.date.toISOString(), arg.allDay);
  };

  const handleEventClick = (arg: EventClickArg) => {
    const schedule = calendar.schedules.find((item) => item.id === arg.event.id);
    if (schedule) {
      setEditing(schedule);
      setDefaultStart(null);
      setDefaultAllDay(false);
      setModalOpened(true);
    }
  };

  useNoticeFocus<Schedule>(
    async (id) => calendar.schedules.find((item) => item.id === id) ?? (await getSchedule(id).catch(() => null)),
    (schedule) => {
      setEditing(schedule);
      setDefaultStart(null);
      setDefaultAllDay(false);
      setModalOpened(true);
    },
  );

  const handleEventChange = async (arg: EventChangeArg) => {
    if (!arg.event.start) {
      arg.revert();
      return;
    }
    try {
      await calendar.update(arg.event.id, {
        startAt: arg.event.start.toISOString(),
        endAt: arg.event.end ? arg.event.end.toISOString() : undefined,
        allDay: arg.event.allDay,
      });
    } catch (cause) {
      arg.revert();
      notifications.show({
        title: t('calendarPage.updateFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    }
  };

  const handleDelete = async (schedule: Schedule) => {
    setDeleting(true);
    try {
      await calendar.remove(schedule.id);
      notifications.show({
        title: t('calendarPage.deleted'),
        message: schedule.title,
        color: 'teal',
      });
      closeModal();
    } catch (cause) {
      notifications.show({
        title: t('calendarPage.deleteFailed'),
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

  const locale = i18n.resolvedLanguage?.startsWith('vi') ? viLocale : undefined;
  const ready = !error && session.profile !== null;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" pb="sm" className={classes.headerBar}>
        <Group gap={5} wrap="nowrap">
          <Text fz={12} c={tokens.textMuted}>
            {t('calendarPage.breadcrumb')}
          </Text>
          <IconChevronRight size={13} style={{ color: tokens.textFaint }} />
          <Text fz={12} fw={600} c={tokens.text}>
            {t('calendarPage.title')}
          </Text>
        </Group>

        <Group justify="space-between" align="center" mt={10} wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconCalendarMonth size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('calendarPage.title')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('calendarPage.subtitle')}
              </Text>
            </div>
          </Group>

          <Button
            leftSection={<IconPlus size={15} />}
            disabled={!session.profile}
            onClick={() => openCreate(new Date().toISOString(), false)}
          >
            {t('calendarPage.newSchedule')}
          </Button>
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} px="lg" pb="lg">
        {error ? <ApiErrorAlert message={error} onRetry={refresh} /> : null}
        {!error && loading && calendar.schedules.length === 0 ? <LoadingBlock /> : null}

        {!error && !loading && !session.profile ? (
          <CenteredPanel
            icon={IconCalendarMonth}
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
          <Box className={classes.calendarWrap}>
            <FullCalendar
              plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
              initialView="dayGridMonth"
              headerToolbar={{
                left: 'prev,next today',
                center: 'title',
                right: 'dayGridMonth,timeGridWeek,timeGridDay',
              }}
              locale={locale}
              height="100%"
              expandRows
              nowIndicator
              dayMaxEvents
              selectable
              editable
              events={events}
              datesSet={handleDatesSet}
              dateClick={handleDateClick}
              eventClick={handleEventClick}
              eventChange={handleEventChange}
            />
          </Box>
        ) : null}
      </Box>

      <ScheduleFormModal
        opened={modalOpened}
        onClose={closeModal}
        schedule={editing}
        defaultStart={defaultStart}
        defaultAllDay={defaultAllDay}
        onDelete={handleDelete}
        deleting={deleting}
      />
    </Flex>
  );
}

export default CalendarPage;
