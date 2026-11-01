/** Quy tắc lặp (RRULE rút gọn) của task: dựng từ preset, nhận lại preset, mô tả bằng lời. */

export type RepeatPreset = 'none' | 'daily' | 'weekdays' | 'weekly' | 'monthly' | 'yearly' | 'custom';
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

/** Ngày hạn (YYYY-MM-DD của form hoặc ISO từ API) -> Date theo giờ địa phương. */
export function dueDate(due: Date | string | null | undefined): Date | null {
  if (!due) {
    return null;
  }
  const date = typeof due === 'string' ? new Date(/^\d{4}-\d{2}-\d{2}$/.test(due) ? `${due}T00:00:00` : due) : due;
  return Number.isNaN(date.getTime()) ? null : date;
}

/** RRULE cho preset; preset theo thứ/ngày lấy từ hạn. null = không lặp hoặc thiếu hạn. */
export function ruleFor(preset: RepeatPreset, due: Date | string | null | undefined, custom = DEFAULT_CUSTOM): string | null {
  const date = dueDate(due);
  if (preset === 'none' || !date) {
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

/** Preset tương ứng một quy tắc có sẵn (để hiện lại trong form). */
export function presetOf(
  rule: string | null | undefined,
  due: Date | string | null | undefined,
): { preset: RepeatPreset; custom: RepeatCustom } {
  if (!rule) {
    return { preset: 'none', custom: DEFAULT_CUSTOM };
  }
  for (const preset of ['daily', 'weekdays', 'weekly', 'monthly', 'yearly'] as const) {
    if (ruleFor(preset, due) === rule.toUpperCase()) {
      return { preset, custom: DEFAULT_CUSTOM };
    }
  }
  const parts = parseRule(rule);
  const unit = (['DAILY', 'WEEKLY', 'MONTHLY'] as const).find((value) => value === parts.FREQ) ?? 'DAILY';
  return { preset: 'custom', custom: { interval: Number(parts.INTERVAL ?? 1) || 1, unit } };
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

/** Mô tả ngắn: "Hằng ngày", "Mỗi Thứ Sáu", "Mỗi 2 tuần", "Hằng tháng ngày 5"… */
export function describeRule(rule: string | null | undefined, t: Translate, language: string): string | null {
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
