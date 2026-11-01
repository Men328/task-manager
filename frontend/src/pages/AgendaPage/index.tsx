import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { Badge, Box, Button, Flex, Group, Stack, Tabs, Text, ThemeIcon, UnstyledButton } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconCalendarDue, IconChevronDown, IconChevronRight, IconSunHigh } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { getTask } from '../../api/task';
import AgendaRow from '../../components/agenda/AgendaRow';
import type { AgendaRowMode } from '../../components/agenda/AgendaRow';
import ProjectFilterChips from '../../components/agenda/ProjectFilterChips';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import QuickAddInput from '../../components/task/QuickAddInput';
import type { QuickAddPrefill } from '../../components/task/QuickAddInput';
import TaskDetailModal from '../../components/task/TaskDetailModal';
import TaskFormModal from '../../components/task/TaskFormModal';
import { useAppDispatch, useProject, useSession } from '../../context';
import { fetchBoard } from '../../context/board/boardSlice';
import { agendaRanges, localDayKey } from '../../lib/dateRanges';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import type { Task } from '../../types';
import classes from './AgendaPage.module.css';
import { useAgenda } from './useAgenda';
import type { AgendaTab } from './useAgenda';

const HIDDEN_PROJECTS_KEY = 'agenda.hiddenProjects';
const COMPLETE_FADE_MS = 1500;

function readHiddenProjects(): Set<string> {
  try {
    const raw = window.localStorage.getItem(HIDDEN_PROJECTS_KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return new Set(Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === 'string') : []);
  } catch {
    return new Set();
  }
}

function writeHiddenProjects(hidden: Set<string>) {
  try {
    window.localStorage.setItem(HIDDEN_PROJECTS_KEY, JSON.stringify([...hidden]));
  } catch {
    // Không lưu được (private mode...) thì bộ lọc chỉ sống trong phiên này.
  }
}

function countByProject(tasks: Task[]): Map<string, number> {
  const counts = new Map<string, number>();
  tasks.forEach((task) => counts.set(task.projectId, (counts.get(task.projectId) ?? 0) + 1));
  return counts;
}

/** Màn Hôm nay / Sắp tới / Quá hạn: gộp task có hạn của mọi project đang hoạt động. */
export function AgendaPage() {
  const { t, i18n } = useTranslation();
  const dispatch = useAppDispatch();
  const { profileId } = useSession();
  const { projects, setCurrent } = useProject();
  const agenda = useAgenda(profileId);

  const [tab, setTab] = useState<AgendaTab>('today');
  const [hidden, setHidden] = useState<Set<string>>(readHiddenProjects);
  const [overdueOpen, setOverdueOpen] = useState(true);
  const [completing, setCompleting] = useState<Set<string>>(new Set());
  const [detailTask, setDetailTask] = useState<Task | null>(null);
  const [detailOpened, setDetailOpened] = useState(false);
  const [formPrefill, setFormPrefill] = useState<QuickAddPrefill | undefined>(undefined);
  const [formOpened, setFormOpened] = useState(false);
  const defaultProjectId = projects.find((project) => project.isDefault)?.id ?? projects[0]?.id ?? null;

  useEffect(() => writeHiddenProjects(hidden), [hidden]);

  const projectById = useMemo(() => new Map(projects.map((project) => [project.id, project])), [projects]);
  const { today0 } = agendaRanges();

  // Chỉ projects đang hoạt động được tải: task của project đã lưu trữ (không có trong map) bị ẩn.
  const inActiveProject = (task: Task) => projectById.has(task.projectId);
  const visible = (task: Task) => inActiveProject(task) && !hidden.has(task.projectId);

  const lists: Record<AgendaTab, Task[]> = {
    overdue: agenda.overdue.filter(inActiveProject),
    today: agenda.today.filter(inActiveProject),
    upcoming: agenda.upcoming.filter(inActiveProject),
  };
  const archivedHidden =
    agenda.overdue.length + agenda.today.length + agenda.upcoming.length -
    (lists.overdue.length + lists.today.length + lists.upcoming.length);

  const tabTasks = tab === 'today' ? [...lists.overdue, ...lists.today] : lists[tab];
  const counts = countByProject(tabTasks);
  const shown = {
    overdue: lists.overdue.filter(visible),
    today: lists.today.filter(visible),
    upcoming: lists.upcoming.filter(visible),
  };

  const toggleProject = (projectId: string) =>
    setHidden((prev) => {
      const next = new Set(prev);
      if (next.has(projectId)) {
        next.delete(projectId);
      } else {
        next.add(projectId);
      }
      return next;
    });

  const openTask = async (task: Task) => {
    // Chuyển switcher sang project của task để sang Board là đúng chỗ; nạp board để modal có status/task cha.
    setCurrent(task.projectId);
    if (profileId) {
      void dispatch(fetchBoard({ profileId, projectId: task.projectId }));
    }
    setDetailTask(task);
    setDetailOpened(true);
    const full = await getTask(task.id).catch(() => null);
    if (full) {
      setDetailTask(full);
    }
  };

  const closeDetail = () => {
    setDetailOpened(false);
    void agenda.reload();
  };

  const blockedReason = (task: Task): string | null => {
    if (!agenda.hasDoneStatus) {
      return t('agendaPage.noDoneStatus');
    }
    if (!agenda.doneTargetFor(task)) {
      return t('agendaPage.noDirectDone', { status: agenda.statusName(task.statusId) });
    }
    return null;
  };

  const undo = async (task: Task, previousStatusId: string) => {
    try {
      await agenda.restoreStatus(task, previousStatusId);
      await agenda.reload();
    } catch (cause) {
      notifications.show({ title: t('agendaPage.undoFailed'), message: getErrorMessage(cause), color: 'red' });
    }
  };

  const completeTask = async (task: Task) => {
    setCompleting((prev) => new Set(prev).add(task.id));
    try {
      const target = await agenda.complete(task);
      const canUndo = agenda.canTransition(target.id, task.statusId);
      const noticeId = `agenda-done-${task.id}`;
      notifications.show({
        id: noticeId,
        color: 'teal',
        autoClose: 6000,
        message: (
          <Group justify="space-between" wrap="nowrap" gap="sm">
            <Text fz={13} truncate>
              {t('agendaPage.doneToast', { title: task.title })}
            </Text>
            {canUndo ? (
              <Button
                size="compact-xs"
                variant="light"
                color="teal"
                onClick={() => {
                  notifications.hide(noticeId);
                  void undo(task, task.statusId);
                }}
              >
                {t('agendaPage.undo')}
              </Button>
            ) : null}
          </Group>
        ),
      });
      window.setTimeout(() => {
        void agenda.reload().finally(() =>
          setCompleting((prev) => {
            const next = new Set(prev);
            next.delete(task.id);
            return next;
          }),
        );
      }, COMPLETE_FADE_MS);
    } catch (cause) {
      setCompleting((prev) => {
        const next = new Set(prev);
        next.delete(task.id);
        return next;
      });
      notifications.show({ title: t('agendaPage.completeFailed'), message: getErrorMessage(cause), color: 'red' });
    }
  };

  const renderRows = (tasks: Task[], mode: AgendaRowMode) => (
    <Stack gap={6}>
      {tasks.map((task) => (
        <AgendaRow
          key={task.id}
          task={task}
          project={projectById.get(task.projectId)}
          parentTitle={task.parentTaskId ? agenda.parentTitles.get(task.parentTaskId) : undefined}
          mode={mode}
          today0={today0}
          completing={completing.has(task.id)}
          completeBlockedReason={blockedReason(task)}
          onComplete={completeTask}
          onOpen={openTask}
        />
      ))}
    </Stack>
  );

  const sectionTitle = (label: string, count: number, extra?: ReactNode) => (
    <Group gap={8} mb={8} mt={4}>
      <Text fz={12} fw={700} tt="uppercase" c={tokens.textMuted}>
        {label}
      </Text>
      <Badge size="sm" variant="light" color="gray">
        {count}
      </Badge>
      {extra}
    </Group>
  );

  const upcomingByDay = new Map<string, Task[]>();
  shown.upcoming.forEach((task) => {
    const key = localDayKey(new Date(task.dueAt as string));
    upcomingByDay.set(key, [...(upcomingByDay.get(key) ?? []), task]);
  });
  const upcomingGroups = [...upcomingByDay.entries()];

  const dayLabel = (key: string) => {
    const date = new Date(`${key}T00:00:00`);
    if (key === localDayKey(agendaRanges().tomorrow0)) {
      return t('agendaPage.tomorrow');
    }
    return date.toLocaleDateString(i18n.language, { weekday: 'long', day: 'numeric', month: 'numeric' });
  };

  const hiddenByFilter = tabTasks.length > 0 && tabTasks.every((task) => hidden.has(task.projectId));
  const loadingFirst = agenda.loading && tabTasks.length === 0 && !agenda.error;

  let body: ReactNode;
  if (agenda.error) {
    body = <ApiErrorAlert message={agenda.error} onRetry={() => void agenda.reload()} />;
  } else if (loadingFirst) {
    body = <LoadingBlock />;
  } else if (hiddenByFilter) {
    body = (
      <CenteredPanel
        title={t('agendaPage.allHiddenTitle')}
        description={t('agendaPage.allHiddenDesc')}
        action={<Button variant="light" onClick={() => setHidden(new Set())}>{t('agendaPage.allProjects')}</Button>}
      />
    );
  } else if (tab === 'today') {
    body =
      shown.overdue.length === 0 && shown.today.length === 0 ? (
        <CenteredPanel
          icon={IconSunHigh}
          title={t('agendaPage.emptyTodayTitle')}
          description={t('agendaPage.emptyTodayDesc')}
          action={<Button variant="light" onClick={() => setTab('upcoming')}>{t('agendaPage.seeUpcoming')}</Button>}
        />
      ) : (
        <Stack gap="lg">
          {shown.overdue.length > 0 ? (
            <Box>
              <UnstyledButton className={classes.sectionToggle} onClick={() => setOverdueOpen((open) => !open)}>
                {overdueOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
                {sectionTitle(t('agendaPage.overdueSection'), shown.overdue.length)}
              </UnstyledButton>
              {overdueOpen ? renderRows(shown.overdue, 'overdue') : null}
            </Box>
          ) : null}
          <Box>
            {sectionTitle(t('agendaPage.todaySection'), shown.today.length)}
            {shown.today.length > 0 ? (
              renderRows(shown.today, 'today')
            ) : (
              <Text fz={13} c={tokens.textMuted}>
                {t('agendaPage.nothingToday')}
              </Text>
            )}
          </Box>
        </Stack>
      );
  } else if (tab === 'upcoming') {
    body =
      upcomingGroups.length === 0 ? (
        <CenteredPanel
          icon={IconCalendarDue}
          title={t('agendaPage.emptyUpcomingTitle')}
          description={t('agendaPage.emptyUpcomingDesc')}
        />
      ) : (
        <Stack gap="lg">
          {upcomingGroups.map(([key, tasks]) => (
            <Box key={key}>
              {sectionTitle(dayLabel(key), tasks.length)}
              {renderRows(tasks, 'upcoming')}
            </Box>
          ))}
        </Stack>
      );
  } else {
    body =
      shown.overdue.length === 0 ? (
        <CenteredPanel
          icon={IconSunHigh}
          title={t('agendaPage.emptyOverdueTitle')}
          description={t('agendaPage.emptyOverdueDesc')}
        />
      ) : (
        renderRows(shown.overdue, 'overdue')
      );
  }

  const tabLabel = (value: AgendaTab, label: string, count: number) => (
    <Tabs.Tab
      value={value}
      rightSection={
        count > 0 ? (
          <Badge size="xs" variant="light" color={value === 'overdue' ? 'red' : 'brand'}>
            {count}
          </Badge>
        ) : null
      }
    >
      {label}
    </Tabs.Tab>
  );

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" className={classes.headerBar}>
        <Group gap={9} wrap="nowrap">
          <ThemeIcon size={30} radius={9} variant="light" color="brand">
            <IconSunHigh size={17} />
          </ThemeIcon>
          <div>
            <Text fz={19} fw={800} c={tokens.text}>
              {t('agendaPage.title')}
            </Text>
            <Text fz={12} c={tokens.textMuted}>
              {t('agendaPage.subtitle')}
            </Text>
          </div>
        </Group>

        <Tabs value={tab} onChange={(value) => value && setTab(value as AgendaTab)} mt="sm">
          <Tabs.List style={{ borderBottom: 0 }}>
            {tabLabel('today', t('agendaPage.tabToday'), lists.today.filter(visible).length)}
            {tabLabel('upcoming', t('agendaPage.tabUpcoming'), lists.upcoming.filter(visible).length)}
            {tabLabel('overdue', t('agendaPage.tabOverdue'), lists.overdue.filter(visible).length)}
          </Tabs.List>
        </Tabs>
      </Box>

      <Box p="lg" className={`${scrollClasses.scroll} ${classes.scrollArea}`}>
        <Stack gap="md" className={classes.content}>
          <QuickAddInput
            defaultProjectId={defaultProjectId}
            onCreated={() => void agenda.reload()}
            onOpenTask={(task) => void openTask(task)}
            onOpenForm={(prefill) => {
              setFormPrefill(prefill);
              setFormOpened(true);
            }}
          />
          <ProjectFilterChips
            projects={projects}
            counts={counts}
            hidden={hidden}
            onToggle={toggleProject}
            onShowAll={() => setHidden(new Set())}
          />
          {body}
          {archivedHidden > 0 && !agenda.error ? (
            <Text fz={12} c={tokens.textFaint}>
              {t('agendaPage.archivedHidden', { count: archivedHidden })}
            </Text>
          ) : null}
        </Stack>
      </Box>

      <TaskDetailModal opened={detailOpened} onClose={closeDetail} task={detailTask} />
      <TaskFormModal
        opened={formOpened}
        onClose={() => setFormOpened(false)}
        prefill={formPrefill}
        onCreated={() => void agenda.reload()}
      />
    </Flex>
  );
}

export default AgendaPage;
