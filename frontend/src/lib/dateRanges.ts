/** Mốc ngày theo múi giờ trình duyệt, dùng cho các màn lọc theo hạn (agenda, board). */

const DAY_MS = 24 * 60 * 60 * 1000;

/** 0h (giờ địa phương) của ngày chứa `date`. */
export function startOfDay(date: Date): Date {
  const out = new Date(date);
  out.setHours(0, 0, 0, 0);
  return out;
}

/** Cộng `days` ngày theo lịch (giữ 0h kể cả khi đổi giờ mùa hè). */
export function addDays(date: Date, days: number): Date {
  const out = new Date(date);
  out.setDate(out.getDate() + days);
  return out;
}

export interface AgendaRanges {
  /** 0h hôm nay. */
  today0: Date;
  /** 0h ngày mai. */
  tomorrow0: Date;
  /** 0h của ngày thứ 8 tính từ hôm nay: "Sắp tới" = [tomorrow0, week0) gồm 7 ngày. */
  week0: Date;
}

export function agendaRanges(now: Date = new Date()): AgendaRanges {
  const today0 = startOfDay(now);
  return { today0, tomorrow0: addDays(today0, 1), week0: addDays(today0, 8) };
}

/** Khoá "YYYY-MM-DD" theo giờ địa phương, để nhóm task theo ngày. */
export function localDayKey(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** Số ngày lịch từ ngày của `from` tới ngày của `to` (to sau from -> dương). */
export function calendarDaysBetween(from: Date, to: Date): number {
  return Math.round((startOfDay(to).getTime() - startOfDay(from).getTime()) / DAY_MS);
}

/** true nếu thời điểm đúng 0h địa phương — quy ước "chỉ có ngày" của form task. */
export function isLocalMidnight(date: Date): boolean {
  return date.getHours() === 0 && date.getMinutes() === 0 && date.getSeconds() === 0;
}

/**
 * Ngày hạn từ form (YYYY-MM-DD -> 0h giờ địa phương) hoặc từ API (ISO) -> Date; null nếu rỗng/không hợp lệ.
 */
export function parseDueDate(due: Date | string | null | undefined): Date | null {
  if (!due) {
    return null;
  }
  const date = typeof due === 'string' ? new Date(/^\d{4}-\d{2}-\d{2}$/.test(due) ? `${due}T00:00:00` : due) : due;
  return Number.isNaN(date.getTime()) ? null : date;
}
