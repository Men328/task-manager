import { useState } from 'react';
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
  Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconArchive, IconArchiveOff, IconFolders, IconPencil, IconPlus } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { getErrorMessage } from '../../api/client';
import { ApiErrorAlert, CenteredPanel, LoadingBlock } from '../../components/common/States';
import { ProjectFormModal } from '../../components/project/ProjectFormModal';
import { useProject } from '../../context';
import scrollClasses from '../../styles/scroll.module.css';
import { projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Project } from '../../types';
import classes from './ProjectsPage.module.css';

type Tab = 'active' | 'archived';

export function ProjectsPage() {
  const { t } = useTranslation();
  const { projects, loading, error, reload, setArchived } = useProject();
  const [tab, setTab] = useState<Tab>('active');
  const [editing, setEditing] = useState<Project | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  const rows = projects.filter((item) => item.isArchived === (tab === 'archived'));

  const openForm = (project: Project | null) => {
    setEditing(project);
    setFormOpen(true);
  };

  const toggleArchived = async (project: Project) => {
    setBusyId(project.id);
    try {
      await setArchived(project.id, !project.isArchived);
    } catch (cause) {
      notifications.show({
        title: t('project.archiveFailed'),
        message: getErrorMessage(cause),
        color: 'red',
      });
    } finally {
      setBusyId(null);
    }
  };

  return (
    <Flex direction="column" h="100%" className={classes.root}>
      <Box px="lg" pt="md" className={classes.headerBar}>
        <Group justify="space-between" align="center" mb="md" wrap="wrap" gap="sm">
          <Group gap={9} wrap="nowrap">
            <ThemeIcon size={30} radius={9} variant="light" color="brand">
              <IconFolders size={17} />
            </ThemeIcon>
            <div>
              <Text fz={19} fw={800} c={tokens.text}>
                {t('project.pageTitle')}
              </Text>
              <Text fz={12} c={tokens.textMuted}>
                {t('project.pageSubtitle')}
              </Text>
            </div>
          </Group>
          <Group gap="sm">
            <SegmentedControl
              value={tab}
              onChange={(value) => setTab(value as Tab)}
              data={[
                { value: 'active', label: t('project.tabActive') },
                { value: 'archived', label: t('project.tabArchived') },
              ]}
            />
            <Button leftSection={<IconPlus size={15} />} onClick={() => openForm(null)}>
              {t('project.createAction')}
            </Button>
          </Group>
        </Group>
      </Box>

      <Box className={`${scrollClasses.scroll} ${classes.scrollArea}`} p="lg">
        {error ? <ApiErrorAlert message={error} onRetry={reload} /> : null}
        {!error && loading && projects.length === 0 ? <LoadingBlock /> : null}

        {!error && !(loading && projects.length === 0) && rows.length === 0 ? (
          <CenteredPanel
            icon={IconFolders}
            title={t('project.emptyTitle')}
            description={tab === 'active' ? t('project.emptyActive') : t('project.emptyArchived')}
          />
        ) : null}

        {!error && rows.length > 0 ? (
          <Box className={classes.panel}>
            <Table verticalSpacing="sm" highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th w={50}>{t('project.colColor')}</Table.Th>
                  <Table.Th>{t('project.colName')}</Table.Th>
                  <Table.Th>{t('project.colDescription')}</Table.Th>
                  <Table.Th w={110} />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {rows.map((project) => (
                  <Table.Tr key={project.id}>
                    <Table.Td>
                      <div className={classes.dot} style={{ background: projectColor(project) }} />
                    </Table.Td>
                    <Table.Td>
                      <Group gap={8} wrap="nowrap">
                        <Text fz={13.5} fw={700} c={tokens.text}>
                          {project.name}
                        </Text>
                        {project.isDefault ? (
                          <Badge size="sm" variant="light" color="brand">
                            {t('project.badgeDefault')}
                          </Badge>
                        ) : null}
                        {project.isArchived ? (
                          <Badge size="sm" variant="light" color="gray">
                            {t('project.badgeArchived')}
                          </Badge>
                        ) : null}
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Text fz={12.5} c={tokens.textMuted} lineClamp={2} maw={520}>
                        {project.description || '—'}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4} justify="flex-end" wrap="nowrap">
                        <Tooltip label={t('project.edit')}>
                          <ActionIcon
                            variant="subtle"
                            color="gray"
                            aria-label={t('project.edit')}
                            onClick={() => openForm(project)}
                          >
                            <IconPencil size={16} />
                          </ActionIcon>
                        </Tooltip>
                        {project.isArchived ? (
                          <Tooltip label={t('project.unarchive')}>
                            <ActionIcon
                              variant="subtle"
                              color="teal"
                              aria-label={t('project.unarchive')}
                              loading={busyId === project.id}
                              onClick={() => void toggleArchived(project)}
                            >
                              <IconArchiveOff size={16} />
                            </ActionIcon>
                          </Tooltip>
                        ) : !project.isDefault ? (
                          <Tooltip label={t('project.archive')}>
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              aria-label={t('project.archive')}
                              loading={busyId === project.id}
                              onClick={() => void toggleArchived(project)}
                            >
                              <IconArchive size={16} />
                            </ActionIcon>
                          </Tooltip>
                        ) : null}
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Box>
        ) : null}
      </Box>

      <ProjectFormModal opened={formOpen} onClose={() => setFormOpen(false)} project={editing} />
    </Flex>
  );
}

export default ProjectsPage;
