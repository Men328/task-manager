import { useEffect, useMemo, useState } from 'react';
import {
  ActionIcon,
  Box,
  Group,
  Loader,
  Select,
  Text,
  ThemeIcon,
  Tooltip,
  UnstyledButton,
} from '@mantine/core';
import { IconPencil, IconPlus } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import {
  useAppDispatch,
  useSession,
  useSidebarCounts,
  useWorkspace,
} from '../../context';
import { fetchWorkspaces } from '../../context/workspace/workspaceSlice';
import { tokens } from '../../theme';
import classes from './WorkspaceSwitcher.module.css';
import { WorkspaceFormModal } from './WorkspaceFormModal';
import type { WorkspaceFormMode } from './WorkspaceFormModal';

export function WorkspaceSwitcher() {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { profileId } = useSession();
  const { tasks, statuses } = useSidebarCounts();
  const { workspaces, selected, selectedId, loading, error, select } = useWorkspace();

  const [formMode, setFormMode] = useState<WorkspaceFormMode | null>(null);

  useEffect(() => {
    if (profileId) {
      void dispatch(fetchWorkspaces(profileId));
    }
  }, [profileId, dispatch]);

  const options = useMemo(
    () => workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name })),
    [workspaces],
  );

  const hasWorkspace = workspaces.length > 0;

  return (
    <Box px={11} py={10} className={classes.spaceBox}>
      <Group justify="space-between" wrap="nowrap" mb={7}>
        <Text fz={10} fw={700} tt="uppercase" c={tokens.textFaint} className={classes.label}>
          {t('workspace.label')}
        </Text>
        {hasWorkspace ? (
          <Group gap={2} wrap="nowrap">
            <Tooltip label={t('workspace.create')} withArrow>
              <ActionIcon
                size={22}
                radius={7}
                variant="subtle"
                color="brand"
                onClick={() => setFormMode('create')}
                aria-label={t('workspace.create')}
              >
                <IconPlus size={14} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label={t('workspace.edit')} withArrow>
              <ActionIcon
                size={22}
                radius={7}
                variant="subtle"
                color="gray"
                onClick={() => setFormMode('edit')}
                disabled={!selected}
                aria-label={t('workspace.edit')}
              >
                <IconPencil size={13} />
              </ActionIcon>
            </Tooltip>
          </Group>
        ) : null}
      </Group>

      {hasWorkspace ? (
        <>
          <Select
            size="xs"
            data={options}
            value={selectedId}
            onChange={(value) => {
              if (value) {
                select(value);
              }
            }}
            placeholder={t('workspace.placeholder')}
            allowDeselect={false}
            checkIconPosition="right"
            aria-label={t('workspace.label')}
          />
          <Text fz={11} c={tokens.textMuted} mt={7} truncate>
            {t('sidebar.spaceMeta', { tasks, statuses })}
          </Text>
        </>
      ) : loading ? (
        <Group gap={6} wrap="nowrap">
          <Loader size={12} color="brand" />
          <Text fz={11.5} c={tokens.textMuted}>
            {t('states.loading')}
          </Text>
        </Group>
      ) : (
        <UnstyledButton
          className={classes.createButton}
          onClick={() => setFormMode('create')}
          disabled={!profileId}
        >
          <ThemeIcon size={30} radius={9} variant="light" color="brand">
            <IconPlus size={17} />
          </ThemeIcon>
          <div className={classes.createBody}>
            <Text fz={12.5} fw={700} c={tokens.text}>
              {t('workspace.createFirst')}
            </Text>
            <Text fz={11} c={tokens.textMuted}>
              {t('workspace.createHint')}
            </Text>
          </div>
        </UnstyledButton>
      )}

      {error && !hasWorkspace ? (
        <Text fz={11} c={tokens.danger} mt={6}>
          {error}
        </Text>
      ) : null}

      <WorkspaceFormModal
        opened={formMode !== null}
        mode={formMode ?? 'create'}
        onClose={() => setFormMode(null)}
        workspace={formMode === 'edit' ? selected : null}
      />
    </Box>
  );
}

export default WorkspaceSwitcher;
