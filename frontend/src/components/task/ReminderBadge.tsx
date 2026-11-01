import { Group, Text, Tooltip } from '@mantine/core';
import { IconBell } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { hasText } from '../../lib/format';
import { tokens } from '../../theme';

/** Chuông + giờ nhắc; chỉ hiện khi nhắc còn ở tương lai. */
export function ReminderBadge({ remindAt }: { remindAt?: string | null }) {
  const { t, i18n } = useTranslation();
  if (!hasText(remindAt)) {
    return null;
  }
  const date = new Date(remindAt as string);
  if (Number.isNaN(date.getTime()) || date.getTime() <= Date.now()) {
    return null;
  }
  const label = date.toLocaleString(i18n.language, { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'numeric' });

  return (
    <Tooltip label={t('reminder.badge', { value: label })} withArrow openDelay={300}>
      <Group gap={3} wrap="nowrap" style={{ flexShrink: 0 }}>
        <IconBell size={12} style={{ color: tokens.brand }} />
        <Text fz={10.5} fw={600} c={tokens.textMuted}>
          {label}
        </Text>
      </Group>
    </Tooltip>
  );
}

export default ReminderBadge;
