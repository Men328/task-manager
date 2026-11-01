/** Quy tắc lặp (RRULE rút gọn) của task: dựng từ preset, nhận lại preset, mô tả bằng lời. */

import { parseDueDate } from './dateRanges';

/** `current` = giữ nguyên quy tắc có sẵn mà các preset không diễn đạt được (COUNT, UNTIL, nhiều thứ…). */
export type RepeatPreset = 'none' | 'daily' | 'weekdays' | 'weekly' | 'monthly' | 'yearly' | 'custom' | 'current';
export type RepeatUnit = 'DAILY' | 'WEEKLY' | 'MONTHLY';

export interface RepeatCustom {
  interval: number;
  unit: RepeatUnit;
}

export const REPEAT_PRESETS: RepeatPreset[] = ['none', 'daily', 'weekdays', 'weekly', 'monthly', 'yearly', 'custom'];
export const DEFAULT_CUSTOM: RepeatCustom = { interval: 2, unit: 'DAILY' };

/** Mã thứ RRULE theo Date.getDay() (0 = CN). */
export const RRULE_DAYS = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA'] as const;
const WEEKDAYS = 'MO,TU,WE,TH,FR';

/** RRULE cho preset; preset theo thứ/ngày lấy từ hạn. null = không lặp hoặc thiếu hạn. */
export function ruleFor(preset: RepeatPreset, due: Date | string | null | undefined, custom = DEFAULT_CUSTOM): string | null {
  const date = parseDueDate(due);
  if (preset === 'none' || preset === 'current' || !date) {
    return null;
  }
  switch (preset) {
    case 'daily':
      return 'FREQ=DAILY';
    case 'weekdays':
      return `FREQ=WEEKLY;BYDAY=${WEEKDAYS}`;
    case 'weekly':
      return `FREQ=WEEKLY;BYDAY=${RRULE_DAYS[date.getDay()]}`;
    case 'monthly':
      return `FREQ=MONTHLY;BYMONTHDAY=${date.getDate()}`;
    case 'yearly':
      return 'FREQ=YEARLY';
    default: {
      const interval = Math.min(365, Math.max(1, Math.round(custom.interval) || 1));
      return interval === 1 ? `FREQ=${custom.unit}` : `FREQ=${custom.unit};INTERVAL=${interval}`;
    }
  }
}

export function parseRule(rule: string | null | undefined): Record<string, string> {
  const parts: Record<string, string> = {};
  (rule ?? '')
    .toUpperCase()
    .split(';')
    .forEach((part) => {
      const [key, value] = part.split('=');
      if (key && value) {
        parts[key.trim()] = value.trim();
      }
    });
  return parts;
}

/** Hai quy tắc tương đương (không phụ thuộc thứ tự thuộc tính; INTERVAL=1 coi như vắng). */
export function rulesEqual(a: string | null | undefined, b: string | null | undefined): boolean {
  const normalize = (rule: string | null | undefined) => {
    const parts = parseRule(rule);
    if (parts.INTERVAL === '1') {
      delete parts.INTERVAL;
    }
    return JSON.stringify(Object.keys(parts).sort().map((key) => [key, parts[key]]));
  };
  return normalize(a) === normalize(b);
}

/**
 * Preset tương ứng một quy tắc có sẵn (để hiện lại trong form). Quy tắc không dựng lại được
 * đúng từ preset nào -> `current`, để lưu không làm mất COUNT/UNTIL/BYDAY… của nó.
 */
export function presetOf(
  rule: string | null | undefined,
  due: Date | string | null | undefined,
): { preset: RepeatPreset; custom: RepeatCustom } {
  if (!rule) {
    return { preset: 'none', custom: DEFAULT_CUSTOM };
  }
  for (const preset of ['daily', 'weekdays', 'weekly', 'monthly', 'yearly'] as const) {
    if (rulesEqual(ruleFor(preset, due), rule)) {
      return { preset, custom: DEFAULT_CUSTOM };
    }
  }
  const parts = parseRule(rule);
  const unit = (['DAILY', 'WEEKLY', 'MONTHLY'] as const).find((value) => value === parts.FREQ);
  const custom = { interval: Number(parts.INTERVAL ?? 1) || 1, unit: unit ?? 'DAILY' };
  if (unit && rulesEqual(ruleFor('custom', due, custom), rule)) {
    return { preset: 'custom', custom };
  }
  return { preset: 'current', custom: DEFAULT_CUSTOM };
}

/** Quy tắc sẽ lưu cho lựa chọn hiện tại; `current` trả lại nguyên quy tắc có sẵn. */
export function resolveRule(
  preset: RepeatPreset,
  due: Date | string | null | undefined,
  custom: RepeatCustom,
  currentRule: string | null | undefined,
): string | null {
  if (preset === 'current') {
    return currentRule || null;
  }
  return ruleFor(preset, due, custom);
}

type Translate = (key: string, options?: Record<string, unknown>) => string;

/** Tên thứ theo ngôn ngữ (2/11/2026 là thứ Hai). */
function dayName(code: string, language: string): string {
  const index = (RRULE_DAYS as readonly string[]).indexOf(code);
  if (index < 0) {
    return code;
  }
  return new Date(2026, 10, 1 + index).toLocaleDateString(language, { weekday: 'long' });
}

/** Mô tả ngắn: "Hằng ngày", "Mỗi Thứ Sáu", "Mỗi 2 tuần", "Hằng tháng ngày 5"… kèm giới hạn COUNT/UNTIL nếu có. */
export function describeRule(rule: string | null | undefined, t: Translate, language: string): string | null {
  const base = describeFrequency(rule, t, language);
  if (!base) {
    return null;
  }
  const parts = parseRule(rule);
  if (parts.COUNT) {
    return `${base} · ${t('repeat.describe.count', { count: Number(parts.COUNT) })}`;
  }
  const until = parts.UNTIL?.match(/^(\d{4})(\d{2})(\d{2})/);
  if (until) {
    const date = new Date(Number(until[1]), Number(until[2]) - 1, Number(until[3]));
    return `${base} · ${t('repeat.describe.until', { date: date.toLocaleDateString(language, { day: 'numeric', month: 'numeric', year: 'numeric' }) })}`;
  }
  return base;
}

function describeFrequency(rule: string | null | undefined, t: Translate, language: string): string | null {
  if (!rule) {
    return null;
  }
  const parts = parseRule(rule);
  const interval = Number(parts.INTERVAL ?? 1) || 1;
  const days = parts.BYDAY ? parts.BYDAY.split(',') : [];
  switch (parts.FREQ) {
    case 'DAILY':
      return interval === 1 ? t('repeat.describe.daily') : t('repeat.describe.everyNDays', { count: interval });
    case 'WEEKLY':
      if (parts.BYDAY === WEEKDAYS) {
        return t('repeat.describe.weekdays');
      }
      if (days.length > 0 && interval === 1) {
        return t('repeat.describe.weeklyOn', { days: days.map((day) => dayName(day, language)).join(', ') });
      }
      return interval === 1 ? t('repeat.describe.weekly') : t('repeat.describe.everyNWeeks', { count: interval });
    case 'MONTHLY':
      if (parts.BYMONTHDAY && interval === 1) {
        return t('repeat.describe.monthlyOn', { day: parts.BYMONTHDAY });
      }
      return interval === 1 ? t('repeat.describe.monthly') : t('repeat.describe.everyNMonths', { count: interval });
    case 'YEARLY':
      return t('repeat.describe.yearly');
    default:
      return t('repeat.describe.other');
  }
}

/** Múi giờ trình duyệt gửi kèm quy tắc lặp. */
export function browserTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Ho_Chi_Minh';
  } catch {
    return 'Asia/Ho_Chi_Minh';
  }
}
