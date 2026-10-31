import { Box, Group, Text, UnstyledButton } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import type { Project } from '../../types';
import { projectColor } from '../../lib/tokens';
import classes from './Agenda.module.css';

interface ProjectFilterChipsProps {
  projects: Project[];
  /** Số task của mỗi project trong tab đang xem (trước khi lọc). */
  counts: Map<string, number>;
  hidden: Set<string>;
  onToggle: (projectId: string) => void;
  onShowAll: () => void;
}

/** Hàng chip bật/tắt từng project; chỉ hiện project có task trong tab đang xem. */
export function ProjectFilterChips({ projects, counts, hidden, onToggle, onShowAll }: ProjectFilterChipsProps) {
  const { t } = useTranslation();
  const withTasks = projects.filter((project) => (counts.get(project.id) ?? 0) > 0);
  if (withTasks.length === 0) {
    return null;
  }
  const allVisible = withTasks.every((project) => !hidden.has(project.id));

  return (
    <Group gap={6} wrap="wrap">
      <UnstyledButton className={classes.chip} data-active={allVisible || undefined} onClick={onShowAll}>
        <Text component="span" fz={12} fw={600}>
          {t('agendaPage.allProjects')}
        </Text>
      </UnstyledButton>
      {withTasks.map((project) => {
        const active = !hidden.has(project.id);
        return (
          <UnstyledButton
            key={project.id}
            className={classes.chip}
            data-active={active || undefined}
            aria-pressed={active}
            onClick={() => onToggle(project.id)}
          >
            <Box component="span" className={classes.dot} style={{ background: projectColor(project) }} />
            <Text component="span" fz={12} fw={600} className={classes.chipName}>
              {project.name}
            </Text>
            <Text component="span" fz={11} className={classes.chipCount}>
              {counts.get(project.id)}
            </Text>
          </UnstyledButton>
        );
      })}
    </Group>
  );
}

export default ProjectFilterChips;
