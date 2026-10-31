import { useState } from 'react';
import { Menu, Text, TextInput, UnstyledButton } from '@mantine/core';
import { IconCheck, IconChevronDown, IconFolders, IconPlus, IconSearch } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { useProject } from '../../context';
import { projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import { ProjectFormModal } from '../project/ProjectFormModal';
import classes from './ProjectSwitcher.module.css';

export function ProjectSwitcher() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { activeProjects, currentProject, currentProjectId, setCurrent } = useProject();
  const [filter, setFilter] = useState('');
  const [creating, setCreating] = useState(false);

  const needle = filter.trim().toLowerCase();
  const visible = needle
    ? activeProjects.filter((item) => item.name.toLowerCase().includes(needle))
    : activeProjects;

  return (
    <>
      <Menu
        position="bottom-start"
        shadow="md"
        width={260}
        radius="md"
        withinPortal
        onClose={() => setFilter('')}
      >
        <Menu.Target>
          <UnstyledButton className={classes.trigger} aria-label={t('project.switcherLabel')}>
            <span className={classes.dot} style={{ background: currentProject ? projectColor(currentProject) : tokens.brand }} />
            <span className={classes.name}>{currentProject?.name ?? t('project.none')}</span>
            <IconChevronDown size={14} className={classes.chevron} />
          </UnstyledButton>
        </Menu.Target>

        <Menu.Dropdown>
          <TextInput
            value={filter}
            onChange={(event) => setFilter(event.currentTarget.value)}
            onKeyDown={(event) => event.stopPropagation()}
            placeholder={t('project.filterPlaceholder')}
            leftSection={<IconSearch size={14} />}
            size="xs"
            mx={6}
            mb={6}
            aria-label={t('project.filterPlaceholder')}
          />
          <div className={classes.list}>
            {visible.length === 0 ? (
              <Text fz={12} c={tokens.textMuted} px="sm" py={6}>
                {t('project.noMatch')}
              </Text>
            ) : (
              visible.map((item) => (
                <Menu.Item
                  key={item.id}
                  onClick={() => setCurrent(item.id)}
                  leftSection={
                    item.id === currentProjectId ? (
                      <IconCheck size={14} className={classes.check} />
                    ) : (
                      <span className={classes.checkPlaceholder} />
                    )
                  }
                >
                  <Text fz={13} truncate>
                    {item.name}
                  </Text>
                </Menu.Item>
              ))
            )}
          </div>
          <Menu.Divider />
          <Menu.Item leftSection={<IconPlus size={14} />} onClick={() => setCreating(true)}>
            {t('project.createAction')}
          </Menu.Item>
          <Menu.Item leftSection={<IconFolders size={14} />} onClick={() => navigate('/projects')}>
            {t('project.manage')}
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>

      <ProjectFormModal
        opened={creating}
        onClose={() => setCreating(false)}
        project={null}
        onCreated={(created) => setCurrent(created.id)}
      />
    </>
  );
}

export default ProjectSwitcher;
