import { useEffect, useState } from 'react';
import { Alert, Box, Button, Flex, Group, Text } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import {
  IconAlertTriangle,
  IconChecklist,
  IconFolderPlus,
  IconPlus,
  IconSparkles,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import BoardHeader from '../../components/board/BoardHeader';
import type { BoardView } from '../../components/board/BoardHeader';
import KanbanBoard from '../../components/board/KanbanBoard';
import TaskTable from '../../components/board/TaskTable';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import TaskFormModal from '../../components/task/TaskFormModal';
import { useAppDispatch, useBoard, useSession, useWorkspace } from '../../context';
import { fetchBoard } from '../../context/board/boardSlice';
import { tokens } from '../../theme';
import classes from './BoardPage.module.css';

export function BoardPage() {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const session = useSession();
  const workspace = useWorkspace();
  const board = useBoard();

  const [view, setView] = useState<BoardView>('kanban');
  const [modalOpened, setModalOpened] = useState(false);
  const [presetStatusId, setPresetStatusId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (session.profileId && workspace.selectedId) {
      void dispatch(
        fetchBoard({ profileId: session.profileId, workspaceId: workspace.selectedId }),
      );
    }
  }, [session.profileId, workspace.selectedId, dispatch]);

  const loading = session.loading || board.loading || workspace.loading;
  const error = session.error ?? board.error;
  const workspaceReady = workspace.selectedId !== null;

  const refresh = () => {
    session.refresh();
    board.refresh();
  };

  const openNewTask = (statusId?: string) => {
    if (!workspaceReady) {
      notifications.show({
        title: t('boardPage.noWorkspaceTitle'),
        message: t('boardPage.noWorkspaceDesc'),
        color: 'yellow',
      });
      return;
    }
    setPresetStatusId(statusId ?? null);
    setModalOpened(true);
  };

  const handleMove = async (taskId: string, statusId: string) => {
    try {
      await board.moveTask(taskId, statusId);
    } catch (cause) {
      notifications.show({
        title: t('boardPage.moveFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
        autoClose: 6000,
      });
    }
  };

  const handleSeed = async () => {
    setBusy(true);
    try {
      await board.seedDefaultStatuses();
      notifications.show({
        title: t('boardPage.seedDoneTitle'),
        message: t('boardPage.seedDoneMessage'),
        color: 'teal',
      });
    } catch (cause) {
      notifications.show({
        title: t('boardPage.seedFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setBusy(false);
    }
  };

  const handleCreateProfile = async () => {
    setBusy(true);
    try {
      await session.createProfile({ email: 'me@example.com', displayName: t('boardPage.defaultProfileName') });
    } catch (cause) {
      notifications.show({
        title: t('boardPage.createProfileFailedTitle'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setBusy(false);
    }
  };

  const hasFilter = session.query.trim().length > 0 || board.priorityFilter.length > 0;
  const ready = !error && session.profile !== null && workspaceReady && board.statuses.length > 0;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <BoardHeader view={view} onViewChange={setView} onNewTask={() => openNewTask()} />

      <Box p="lg" className={classes.content}>
        {error ? <ApiErrorAlert message={error} onRetry={refresh} /> : null}

        {!error && loading && board.statuses.length === 0 ? <LoadingBlock /> : null}

        {!error && !loading && !session.profile ? (
          <CenteredPanel
            icon={IconChecklist}
            title={t('boardPage.noProfileTitle')}
            description={t('boardPage.noProfileDesc')}
            action={
              <Button loading={busy} leftSection={<IconPlus size={15} />} onClick={handleCreateProfile}>
                {t('boardPage.createProfile')}
              </Button>
            }
          />
        ) : null}

        {!error && !loading && session.profile && !workspaceReady ? (
          <CenteredPanel
            icon={IconFolderPlus}
            title={t('boardPage.noWorkspaceTitle')}
            description={t('boardPage.noWorkspaceDesc')}
          />
        ) : null}

        {!error && !loading && session.profile && workspaceReady && board.statuses.length === 0 ? (
          <CenteredPanel
            icon={IconSparkles}
            title={t('boardPage.noStatusTitle')}
            description={t('boardPage.noStatusDesc')}
            action={
              <Button loading={busy} leftSection={<IconSparkles size={15} />} onClick={handleSeed}>
                {t('boardPage.seedStatuses')}
              </Button>
            }
          />
        ) : null}

        {ready && board.tasks.length > 0 && board.visibleTaskCount === 0 && hasFilter ? (
          <Alert
            color="yellow"
            variant="light"
            icon={<IconAlertTriangle size={17} />}
            radius="md"
          >
            <Group justify="space-between" wrap="nowrap">
              <Text fz={13}>{t('boardPage.noMatch')}</Text>
              <Button size="compact-xs" variant="light" color="yellow" onClick={board.clearFilters}>
                {t('boardHeader.clearFilter')}
              </Button>
            </Group>
          </Alert>
        ) : null}

        {ready && view === 'kanban' ? (
          <KanbanBoard onAddTask={openNewTask} onMoveTask={handleMove} />
        ) : null}
        {ready && view === 'table' ? <TaskTable /> : null}
        {ready && (view === 'list' || view === 'timeline') ? (
          <CenteredPanel
            title={view === 'list' ? t('boardPage.listView') : t('boardPage.timelineView')}
            description={
              <Text fz={13} c={tokens.textMuted}>
                {t('boardPage.soonDesc')}
              </Text>
            }
          />
        ) : null}
      </Box>

      <TaskFormModal
        opened={modalOpened}
        onClose={() => setModalOpened(false)}
        presetStatusId={presetStatusId}
      />
    </Flex>
  );
}

export default BoardPage;
