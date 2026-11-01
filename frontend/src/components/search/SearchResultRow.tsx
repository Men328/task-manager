import { Box, Group, Text } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { formatShortDate } from '../../lib/format';
import { highlightParts } from '../../lib/searchText';
import { projectColor } from '../../lib/tokens';
import { tokens } from '../../theme';
import type { Project, Task, TaskStatus } from '../../types';
import classes from './GlobalSearch.module.css';

interface SearchResultRowProps {
  task: Task;
  query: string;
  project: Project | undefined;
  status: TaskStatus | undefined;
}

function isClosed(status: TaskStatus | undefined): boolean {
  return status?.category === 'TASK_STATUS_CATEGORY_DONE' || status?.category === 'TASK_STATUS_CATEGORY_CANCELLED';
}

/** Một kết quả tìm kiếm: chấm màu + tên project, tiêu đề tô đậm đoạn khớp, status, hạn. */
export function SearchResultRow({ task, query, project, status }: SearchResultRowProps) {
  const { t, i18n } = useTranslation();
  const closed = isClosed(status);
  const due = formatShortDate(task.dueAt, i18n.language);

  return (
    <Box className={classes.result} data-closed={closed || undefined}>
      <Text fz={13} fw={600} c={tokens.text} td={closed ? 'line-through' : undefined} lineClamp={1}>
        {highlightParts(task.title, query).map((part, index) =>
          part.match ? (
            <mark key={index} className={classes.mark}>
              {part.text}
            </mark>
          ) : (
            <span key={index}>{part.text}</span>
          ),
        )}
      </Text>
      <Group gap={8} wrap="nowrap" mt={3}>
        <Group gap={5} wrap="nowrap" className={classes.project}>
          <span className={classes.dot} style={{ background: projectColor(project) }} />
          <Text fz={11.5} c={tokens.textMuted} truncate>
            {project?.name ?? t('search.archivedProject')}
          </Text>
        </Group>
        {status ? (
          <Text fz={11.5} c={tokens.textFaint} truncate>
            · {status.name}
          </Text>
        ) : null}
        {due ? (
          <Text fz={11.5} c={tokens.textFaint} className={classes.due}>
            · {due}
          </Text>
        ) : null}
      </Group>
    </Box>
  );
}

export default SearchResultRow;
