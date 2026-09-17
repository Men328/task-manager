import { Box, Group, Text } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { tokens } from '../../theme';
import classes from './TaskProgress.module.css';

/** Thanh tiến độ của card. `value` là % (0-100). */
export function TaskProgress({ value }: { value: number }) {
  const { t } = useTranslation();
  const clamped = Math.max(0, Math.min(100, value));

  return (
    <Box>
      <Group justify="space-between" gap={8} mb={5} wrap="nowrap">
        <Text fz={11.5} fw={600} c={tokens.textMuted}>
          {t('taskCard.progress')}
        </Text>
        <Text fz={11.5} fw={700} c={tokens.text}>
          {clamped}%
        </Text>
      </Group>
      <Box className={classes.track}>
        <Box className={classes.fill} style={{ width: `${clamped}%` }} />
      </Box>
    </Box>
  );
}

export default TaskProgress;
