import { parseDueDate, startOfDay } from './dateRanges';

/** Preset "Nhắc tôi" của task. Giờ nhắc lưu dạng ISO (UTC); preset tính theo giờ địa phương. */

export type ReminderPreset = 'none' | 'dueMorning' | 'dayBefore' | 'custom';

export const REMINDER_PRESETS: ReminderPreset[] = ['none', 'dueMorning', 'dayBefore', 'custom'];

/** Giờ nhắc mặc định của preset theo hạn. */
export const REMINDER_HOUR = 9;

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

/** "YYYY-MM-DDTHH:mm" theo giờ địa phương — giá trị của <input type="datetime-local">. */
export function toLocalInput(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** Ngày hạn (Date, chuỗi YYYY-MM-DD của form, hoặc ISO từ API) -> Date 0h địa phương của ngày đó. */
function dueDay(due: Date | string): Date | null {
  const date = parseDueDate(due);
  return date ? startOfDay(date) : null;
}

/**
 * Thời điểm nhắc theo preset. `dueMorning` = 9h ngày đến hạn, `dayBefore` = 9h ngày trước hạn,
 * `custom` = giá trị datetime-local. Trả null khi không nhắc hoặc thiếu dữ liệu.
 */
export function reminderFor(
  preset: ReminderPreset,
  due: Date | string | null | undefined,
  custom: string,
): Date | null {
  if (preset === 'custom') {
    const date = custom ? new Date(custom) : null;
    return date && !Number.isNaN(date.getTime()) ? date : null;
  }
  if (preset === 'none' || !due) {
    return null;
  }
  const day = dueDay(due);
  if (!day) {
    return null;
  }
  const offset = preset === 'dayBefore' ? -1 : 0;
  return new Date(day.getFullYear(), day.getMonth(), day.getDate() + offset, REMINDER_HOUR, 0);
}

/** Đoán preset của một giờ nhắc có sẵn (để hiển thị lại trong form). */
export function presetOf(remindAt: Date | null, due: Date | string | null | undefined): ReminderPreset {
  if (!remindAt) {
    return 'none';
  }
  for (const preset of ['dueMorning', 'dayBefore'] as const) {
    const candidate = reminderFor(preset, due, '');
    if (candidate && candidate.getTime() === remindAt.getTime()) {
      return preset;
    }
  }
  return 'custom';
}
