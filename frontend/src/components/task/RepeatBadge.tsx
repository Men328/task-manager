import { Group, Text, Tooltip } from '@mantine/core';
import { IconRepeat } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { describeRule } from '../../lib/recurrence';
import { tokens } from '../../theme';

/** Icon lặp + mô tả ngắn ("Mỗi Thứ Sáu"); không hiện gì nếu task không lặp. */
export function RepeatBadge({ rule, compact }: { rule?: string | null; compact?: boolean }) {
  const { t, i18n } = useTranslation();
  const label = describeRule(rule, t, i18n.language);
  if (!label) {
    return null;
  }
  const content = (
    <Group gap={3} wrap="nowrap" style={{ flexShrink: 0 }}>
      <IconRepeat size={12} style={{ color: tokens.brand }} />
      {compact ? null : (
        <Text fz={10.5} fw={600} c={tokens.textMuted} truncate maw={140}>
          {label}
        </Text>
      )}
    </Group>
  );
  return (
    <Tooltip label={t('repeat.badge', { rule: label })} withArrow openDelay={300}>
      {content}
    </Tooltip>
  );
}

export default RepeatBadge;
