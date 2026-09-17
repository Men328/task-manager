import { Box, Text } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { PRIORITY_LABEL_KEY } from '../../lib/tokens';
import type { TaskPriority } from '../../types';
import classes from './PriorityBadge.module.css';

const MODIFIER: Record<TaskPriority, string> = {
  TASK_PRIORITY_UNSPECIFIED: classes.unspecified,
  TASK_PRIORITY_LOW: classes.low,
  TASK_PRIORITY_MEDIUM: classes.medium,
  TASK_PRIORITY_HIGH: classes.high,
  TASK_PRIORITY_URGENT: classes.urgent,
};

export function PriorityBadge({ priority }: { priority: TaskPriority }) {
  const { t } = useTranslation();
  const label = t(PRIORITY_LABEL_KEY[priority] ?? PRIORITY_LABEL_KEY.TASK_PRIORITY_UNSPECIFIED);
  const modifier = MODIFIER[priority] ?? classes.unspecified;

  return (
    <Box component="span" className={`${classes.badge} ${modifier}`}>
      <Text fz={9} fw={800} tt="uppercase" className={classes.label}>
        {label}
      </Text>
    </Box>
  );
}

export default PriorityBadge;
