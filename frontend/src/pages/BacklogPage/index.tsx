import { useEffect, useState } from 'react';
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
import { IconArchive, IconChevronRight, IconPencil, IconPlus } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { getBacklog } from '../../api/backlog';
import BacklogFormModal from '../../components/backlog/BacklogFormModal';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { useBacklog, useNoticeFocus, useSession } from '../../context';
import {
  BACKLOG_CATEGORY_LABEL_KEY,
  BACKLOG_STATUS_COLOR,
  BACKLOG_STATUS_LABEL_KEY,
} from '../../lib/tokens';
import scrollClasses from '../../styles/scroll.module.css';
import { tokens } from '../../theme';
import { BACKLOG_STATUSES, type Backlog, type BacklogStatus } from '../../types';
import classes from './BacklogPage.module.css';

type StatusFilter = 'all' | BacklogStatus;

export function BacklogPage() {
  const { t, i18n } = useTranslation();
  const session = useSession();
  const backlogState = useBacklog();

  const [modalOpened, setModalOpened] = useState(false);
  const [editing, setEditing] = useState<Backlog | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  const { profileId, applyFilter } = backlogState;

  useEffect(() => {
    if (profileId) {
      applyFilter({ status: statusFilter === 'all' ? undefined : statusFilter });
    }
  }, [profileId, statusFilter, applyFilter]);

  const locale = i18n.resolvedLanguage?.startsWith('vi') ? 'vi-VN' : 'en-GB';
  const loading = session.loading || backlogState.loading;
  const error = session.error ?? backlogState.error;

  const refreshAll = () => {
    session.refresh();
    applyFilter({ status: statusFilter === 'all' ? undefined : statusFilter });
  };

  const openCreate = () => {
    setEditing(null);
    setModalOpened(true);
  };

  const openEdit = (item: Backlog) => {
    setEditing(item);
    setModalOpened(true);
  };

  const closeModal = () => {
    setModalOpened(false);
    setEditing(null);
  };

  useNoticeFocus<Backlog>(
    async (id) => backlogState.items.find((item) => item.id === id) ?? (await getBacklog(id).catch(() => null)),
    (item) => openEdit(item),
  );

  const handleDelete = async (item: Backlog) => {
    setDeleting(true);
    try {
      await backlogState.remove(item.id);
      notifications.show({
        title: t('backlogPage.deleted'),
        message: item.title,
        color: 'teal',
      });
      closeModal();
    } catch (cause) {
      notifications.show({
        title: t('backlogPage.deleteFailed'),
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

  const formatDate = (value: string | null | undefined): string => {
    if (!value) {
      return '—';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '—';
    }
    return date.toLocaleString(locale, {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const ready = !error && session.profile !== null;

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" pb="sm" className={classes.headerBar}>
        <Group gap={5} wrap="nowrap">
          <Text fz={12} c={tokens.textMuted}>
            {t('backlogPage.breadcrumb')}
          </Text>
          <IconChevronRight size={13} style={{ color: tokens.textFaint }} />
          <Text fz={12} fw={600} c={tokens.text}>
            {t('backlogPage.title')}
          </Text>
        </Group>

        <Group justify="space-between" align="center" mt={10} wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconArchive size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('backlogPage.title')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('backlogPage.subtitle')}
              </Text>
            </div>
          </Group>

          <Group gap="sm">
            <SegmentedControl
              size="xs"
              value={statusFilter}
              onChange={(value) => setStatusFilter(value as StatusFilter)}
              data={[
                { value: 'all', label: t('backlogPage.filterAll') },
                ...BACKLOG_STATUSES.map((value) => ({
                  value,
                  label: t(BACKLOG_STATUS_LABEL_KEY[value]),
                })),
              ]}
            />
            <Button
              leftSection={<IconPlus size={15} />}
              disabled={!session.profile}
              onClick={openCreate}
            >
              {t('backlogPage.newItem')}
            </Button>
          </Group>
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        {error ? <ApiErrorAlert message={error} onRetry={refreshAll} /> : null}
        {!error && loading && backlogState.items.length === 0 ? <LoadingBlock /> : null}

        {!error && !loading && !session.profile ? (
          <CenteredPanel
            icon={IconArchive}
            title={t('boardPage.noProfileTitle')}
            description={t('boardPage.noProfileDesc')}
            action={
              <Button leftSection={<IconPlus size={15} />} onClick={handleCreateProfile}>
                {t('boardPage.createProfile')}
              </Button>
            }
          />
        ) : null}

        {ready && backlogState.items.length === 0 ? (
          <CenteredPanel
            icon={IconArchive}
            title={t('backlogPage.emptyTitle')}
            description={t('backlogPage.emptyDesc')}
            action={
              <Button leftSection={<IconPlus size={15} />} onClick={openCreate}>
                {t('backlogPage.newItem')}
              </Button>
            }
          />
        ) : null}

        {ready && backlogState.items.length > 0 ? (
          <Box className={classes.panel}>
            <Box px="md" py="sm" className={classes.panelHeader}>
              <Text fz={13.5} fw={700} c={tokens.text}>
                {t('backlogPage.panelTitle', { n: backlogState.items.length })}
              </Text>
              <Text fz={12} c={tokens.textMuted} mt={2}>
                {t('backlogPage.panelDesc')}
              </Text>
            </Box>
            <Table highlightOnHover verticalSpacing="sm" horizontalSpacing="md">
              <Table.Thead>
                <Table.Tr className={classes.headRow}>
                  <Table.Th>{t('backlogPage.colItem')}</Table.Th>
                  <Table.Th w={200}>{t('backlogPage.colSender')}</Table.Th>
                  <Table.Th w={130}>{t('backlogPage.colCategory')}</Table.Th>
                  <Table.Th w={180}>{t('backlogPage.colReason')}</Table.Th>
                  <Table.Th w={150}>{t('backlogPage.colCreated')}</Table.Th>
                  <Table.Th w={130}>{t('backlogPage.colStatus')}</Table.Th>
                  <Table.Th w={70} />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {backlogState.items.map((item) => {
                  const categoryKey = BACKLOG_CATEGORY_LABEL_KEY[item.category];
                  return (
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
                        <Text fz={12.5} c={tokens.textMuted} lineClamp={1}>
                          {item.sender || '—'}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="light" color="grape">
                          {categoryKey ? t(categoryKey) : item.category}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Text fz={12.5} c={tokens.textMuted} lineClamp={1}>
                          {item.reason || '—'}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Text fz={12.5} c={tokens.textMuted}>
                          {formatDate(item.createdAt)}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="light" color={BACKLOG_STATUS_COLOR[item.status]}>
                          {t(BACKLOG_STATUS_LABEL_KEY[item.status])}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <ActionIcon variant="subtle" color="gray" onClick={() => openEdit(item)}>
                          <IconPencil size={15} />
                        </ActionIcon>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Box>
        ) : null}
      </Box>

      <BacklogFormModal
        opened={modalOpened}
        onClose={closeModal}
        item={editing}
        onDelete={handleDelete}
        deleting={deleting}
      />
    </Flex>
  );
}

export default BacklogPage;
