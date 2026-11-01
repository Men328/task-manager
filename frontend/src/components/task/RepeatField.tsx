import { Group, NumberInput, Select, Text } from '@mantine/core';
import { IconRepeat } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

import { REPEAT_PRESETS, describeRule, dueDate, ruleFor } from '../../lib/recurrence';
import type { RepeatCustom, RepeatPreset, RepeatUnit } from '../../lib/recurrence';
import { tokens } from '../../theme';

interface RepeatFieldProps {
  preset: RepeatPreset;
  custom: RepeatCustom;
  /** Hạn của task; lặp cần có hạn. */
  due: Date | string | null | undefined;
  onChange: (preset: RepeatPreset, custom: RepeatCustom) => void;
  disabled?: boolean;
}

/** Ô "Lặp lại": Không · Hằng ngày · Ngày làm việc · Hằng tuần vào [thứ của hạn] · Hằng tháng ngày [ngày của hạn] · Hằng năm · Tuỳ chỉnh. */
export function RepeatField({ preset, custom, due, onChange, disabled }: RepeatFieldProps) {
  const { t, i18n } = useTranslation();
  const date = dueDate(due);
  const rule = ruleFor(preset, due, custom);

  const presetLabel = (value: RepeatPreset) => {
    if (value === 'weekly' && date) {
      return t('repeat.preset_weeklyOn', { day: date.toLocaleDateString(i18n.language, { weekday: 'long' }) });
    }
    if (value === 'monthly' && date) {
      return t('repeat.preset_monthlyOn', { day: date.getDate() });
    }
    return t(`repeat.preset_${value}`);
  };

  return (
    <div>
      <Group grow align="flex-start">
        <Select
          label={t('repeat.label')}
          leftSection={<IconRepeat size={15} style={{ color: tokens.textFaint }} />}
          data={REPEAT_PRESETS.map((value) => ({ value, label: presetLabel(value) }))}
          value={date ? preset : 'none'}
          onChange={(value) => onChange((value as RepeatPreset) ?? 'none', custom)}
          allowDeselect={false}
          disabled={disabled || !date}
        />
        {preset === 'custom' && date ? (
          <Group gap="xs" grow wrap="nowrap" align="flex-end">
            <NumberInput
              label={t('repeat.every')}
              min={1}
              max={365}
              value={custom.interval}
              onChange={(value) => onChange('custom', { ...custom, interval: Number(value) || 1 })}
              disabled={disabled}
            />
            <Select
              aria-label={t('repeat.unit')}
              data={(['DAILY', 'WEEKLY', 'MONTHLY'] as RepeatUnit[]).map((unit) => ({
                value: unit,
                label: t(`repeat.unit_${unit}`),
              }))}
              value={custom.unit}
              onChange={(value) => onChange('custom', { ...custom, unit: (value as RepeatUnit) ?? 'DAILY' })}
              allowDeselect={false}
              disabled={disabled}
            />
          </Group>
        ) : null}
      </Group>
      <Text fz={11.5} c={tokens.textFaint} mt={4}>
        {!date
          ? t('repeat.needDue')
          : rule
            ? t('repeat.summary', { rule: describeRule(rule, t, i18n.language) })
            : null}
      </Text>
    </div>
  );
}

export default RepeatField;
