import { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Button, Flex, Group, Text, ThemeIcon } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconCalendarEvent, IconCalendarMonth, IconChevronRight, IconPlus } from '@tabler/icons-react';
import type { DateClickArg } from '@fullcalendar/interaction';
import interactionPlugin from '@fullcalendar/interaction';
import type { DatesSetArg, EventChangeArg, EventClickArg, EventContentArg, EventInput } from '@fullcalendar/core';
import viLocale from '@fullcalendar/core/locales/vi';
import dayGridPlugin from '@fullcalendar/daygrid';
import FullCalendar from '@fullcalendar/react';
import timeGridPlugin from '@fullcalendar/timegrid';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { getSchedule } from '../../api/calendar';
import ScheduleFormModal from '../../components/calendar/ScheduleFormModal';
import EventFormModal from '../../components/event/EventFormModal';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useCalendar, useEvent, useNoticeFocus, useSession } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { Event, Schedule } from '../../types';
import classes from './CalendarPage.module.css';

/** Loại mục hiển thị trên lịch: lịch thông thường hoặc sự kiện. */
type CalendarItemKind = 'schedule' | 'event';

// Id trên FullCalendar có tiền tố loại để lịch và sự kiện trùng id không đè nhau.
const KIND_SEPARATOR = ':';

function calendarId(kind: CalendarItemKind, id: string): string {
  return `${kind}${KIND_SEPARATOR}${id}`;
}

function parseCalendarId(value: string): { kind: CalendarItemKind; id: string } | null {
  const index = value.indexOf(KIND_SEPARATOR);
  if (index < 0) {
    return null;
  }
  const kind = value.slice(0, index);
  if (kind !== 'schedule' && kind !== 'event') {
    return null;
  }
  return { kind, id: value.slice(index + 1) };
}

export function CalendarPage() {
  const { t, i18n } = useTranslation();
  const session = useSession();
  const calendar = useCalendar();
  const eventState = useEvent();

  const [modalOpened, setModalOpened] = useState(false);
  const [editing, setEditing] = useState<Schedule | null>(null);
  const [defaultStart, setDefaultStart] = useState<string | null>(null);
  const [defaultAllDay, setDefaultAllDay] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [eventModalOpened, setEventModalOpened] = useState(false);
  const [editingEvent, setEditingEvent] = useState<Event | null>(null);
  const [deletingEvent, setDeletingEvent] = useState(false);

  const rangeRef = useRef<{ from: string; to: string } | null>(null);
  const loadedKeyRef = useRef<string | null>(null);
  const { profileId, projectId, loadRange } = calendar;
  const loadKey = profileId && projectId ? `${profileId}:${projectId}` : null;

  useEffect(() => {
    if (loadKey && rangeRef.current && loadedKeyRef.current !== loadKey) {
      loadedKeyRef.current = loadKey;
      loadRange(rangeRef.current.from, rangeRef.current.to);
    }
  }, [loadKey, loadRange]);

  // Sự kiện không lọc theo khoảng ngày ở API: tải toàn bộ sự kiện của project hiện tại.
  const { profileId: eventProfileId, projectId: eventProjectId, refresh: refreshEvents } = eventState;
  useEffect(() => {
    if (eventProfileId && eventProjectId) {
      refreshEvents();
    }
  }, [eventProfileId, eventProjectId, refreshEvents]);

  const events = useMemo<EventInput[]>(
    () => [
      ...calendar.schedules.map((schedule) => ({
        id: calendarId('schedule', schedule.id),
        title: schedule.title,
        start: schedule.startAt,
        end: schedule.endAt ?? undefined,
        allDay: schedule.allDay,
        backgroundColor: schedule.color || undefined,
        borderColor: schedule.color || undefined,
        classNames: [classes.scheduleItem],
        extendedProps: {
          kind: 'schedule' as CalendarItemKind,
          description: schedule.description,
          location: schedule.location,
        },
      })),
      ...eventState.events
        .filter((item) => item.status !== 'EVENT_STATUS_CANCELLED')
        .map((item) => ({
          id: calendarId('event', item.id),
          title: item.title,
          start: item.startAt,
          end: item.endAt ?? undefined,
          allDay: item.allDay,
          backgroundColor: item.color || undefined,
          borderColor: item.color || undefined,
          classNames: [classes.eventItem],
          extendedProps: {
            kind: 'event' as CalendarItemKind,
            description: item.description,
            location: item.location,
          },
        })),
    ],
    [calendar.schedules, eventState.events],
  );

  const loading = session.loading || calendar.loading;
  const error = session.error ?? calendar.error ?? eventState.error;

  const refresh = () => {
    session.refresh();
    calendar.refresh();
    eventState.refresh();
  };

  const openEvent = (item: Event | null) => {
    setEditingEvent(item);
    setEventModalOpened(true);
  };

  const closeEventModal = () => {
    setEventModalOpened(false);
    setEditingEvent(null);
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
    if (loadKey) {
      loadedKeyRef.current = loadKey;
      loadRange(arg.startStr, arg.endStr);
    }
  };

  const handleDateClick = (arg: DateClickArg) => {
    openCreate(arg.date.toISOString(), arg.allDay);
  };

  const handleEventClick = (arg: EventClickArg) => {
    const parsed = parseCalendarId(arg.event.id);
    if (parsed?.kind === 'event') {
      const item = eventState.events.find((candidate) => candidate.id === parsed.id);
      if (item) {
        openEvent(item);
      }
      return;
    }
    const schedule = calendar.schedules.find((item) => item.id === parsed?.id);
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
    (schedule) => schedule.projectId,
  );

  const handleEventChange = async (arg: EventChangeArg) => {
    if (!arg.event.start) {
      arg.revert();
      return;
    }
    const parsed = parseCalendarId(arg.event.id);
    if (!parsed) {
      arg.revert();
      return;
    }
    try {
      const update = parsed.kind === 'event' ? eventState.update : calendar.update;
      await update(parsed.id, {
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

  const handleDeleteEvent = async (item: Event) => {
    setDeletingEvent(true);
    try {
      await eventState.remove(item.id);
      notifications.show({
        title: t('eventPage.deleted'),
        message: item.title,
        color: 'teal',
      });
      closeEventModal();
    } catch (cause) {
      notifications.show({
        title: t('eventPage.deleteFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setDeletingEvent(false);
    }
  };

  const renderItem = (arg: EventContentArg) => {
    const kind = arg.event.extendedProps.kind as CalendarItemKind | undefined;
    const isEvent = kind === 'event';
    return (
      <div className={classes.itemContent}>
        <span className={`${classes.kindTag} ${isEvent ? classes.kindTagEvent : classes.kindTagSchedule}`}>
          {isEvent ? t('calendarPage.tagEvent') : t('calendarPage.tagSchedule')}
        </span>
        {arg.timeText ? <span className={classes.itemTime}>{arg.timeText}</span> : null}
        <span className={classes.itemTitle}>{arg.event.title}</span>
      </div>
    );
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

          <Group gap="sm" wrap="wrap">
            <Group gap={8} wrap="nowrap" className={classes.legend}>
              <span className={`${classes.kindTag} ${classes.kindTagSchedule}`}>{t('calendarPage.tagSchedule')}</span>
              <span className={`${classes.kindTag} ${classes.kindTagEvent}`}>{t('calendarPage.tagEvent')}</span>
            </Group>
            <Button
              variant="light"
              leftSection={<IconCalendarEvent size={15} />}
              disabled={!session.profile}
              onClick={() => openEvent(null)}
            >
              {t('calendarPage.newEvent')}
            </Button>
            <Button
              leftSection={<IconPlus size={15} />}
              disabled={!session.profile}
              onClick={() => openCreate(new Date().toISOString(), false)}
            >
              {t('calendarPage.newSchedule')}
            </Button>
          </Group>
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
              eventContent={renderItem}
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

      <EventFormModal
        opened={eventModalOpened}
        onClose={closeEventModal}
        event={editingEvent}
        onDelete={handleDeleteEvent}
        deleting={deletingEvent}
      />
    </Flex>
  );
}

export default CalendarPage;
