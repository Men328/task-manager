import { Group, Select, Text, TextInput } from '@mantine/core';
import { IconBell } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { REMINDER_PRESETS, reminderFor } from '../../lib/reminders';
import type { ReminderPreset } from '../../lib/reminders';
import { tokens } from '../../theme';

interface ReminderFieldProps {
  preset: ReminderPreset;
  custom: string;
  /** Hạn của task (YYYY-MM-DD hoặc ISO); preset theo hạn chỉ bật khi có hạn. */
  due: Date | string | null | undefined;
  onChange: (preset: ReminderPreset, custom: string) => void;
  disabled?: boolean;
}

/** Ô "Nhắc tôi": Không nhắc · 9h ngày đến hạn · 9h ngày trước hạn · Tuỳ chọn (ngày + giờ). */
export function ReminderField({ preset, custom, due, onChange, disabled }: ReminderFieldProps) {
  const { t, i18n } = useTranslation();
  const hasDue = Boolean(due);
  const resolved = reminderFor(preset, due, custom);
  const inPast = resolved !== null && resolved.getTime() <= Date.now();

  const data = REMINDER_PRESETS.map((value) => ({
    value,
    label: t(`reminder.preset_${value}`),
    disabled: !hasDue && (value === 'dueMorning' || value === 'dayBefore'),
  }));

  return (
    <div>
      <Group grow align="flex-start">
        <Select
          label={t('reminder.label')}
          leftSection={<IconBell size={15} style={{ color: tokens.textFaint }} />}
          data={data}
          value={preset}
          onChange={(value) => onChange((value as ReminderPreset) ?? 'none', custom)}
          allowDeselect={false}
          disabled={disabled}
        />
        {preset === 'custom' ? (
          <TextInput
            label={t('reminder.customLabel')}
            type="datetime-local"
            value={custom}
            onChange={(event) => onChange('custom', event.currentTarget.value)}
            disabled={disabled}
          />
        ) : null}
      </Group>
      {!hasDue && preset === 'none' ? (
        <Text fz={11.5} c={tokens.textFaint} mt={4}>
          {t('reminder.needDue')}
        </Text>
      ) : null}
      {resolved ? (
        <Text fz={11.5} c={inPast ? tokens.danger : tokens.textMuted} mt={4}>
          {inPast
            ? t('reminder.inPast')
            : t('reminder.willRemind', {
                value: resolved.toLocaleString(i18n.language, {
                  weekday: 'short',
                  day: 'numeric',
                  month: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                }),
              })}
        </Text>
      ) : null}
    </div>
  );
}

export default ReminderField;
