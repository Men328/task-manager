/** Tiện ích format dùng chung cho UI. */

/** true nếu chuỗi có nội dung (API trả "" cho field chưa set). */
export function hasText(value: string | null | undefined): boolean {
  return typeof value === 'string' && value.trim().length > 0;
}

/** "12 Sep" theo locale hiện tại — trả null nếu không parse được. */
export function formatShortDate(
  value: string | null | undefined,
  locale = 'en-GB',
): string | null {
  if (!hasText(value)) {
    return null;
  }
  const date = new Date(value as string);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date.toLocaleDateString(locale, { day: '2-digit', month: 'short' });
}

/** Ngày quá hạn (so với hôm nay, bỏ qua giờ). */
export function isOverdue(value: string | null | undefined): boolean {
  if (!hasText(value)) {
    return false;
  }
  const date = new Date(value as string);
  if (Number.isNaN(date.getTime())) {
    return false;
  }
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return date.getTime() < today.getTime();
}

/** "Rico Tandoor" -> "RT" */
export function initials(name: string | null | undefined, fallback = '?'): string {
  if (!hasText(name)) {
    return fallback;
  }
  const parts = (name as string).trim().split(/\s+/).slice(0, 2);
  return parts.map((part) => part.charAt(0).toUpperCase()).join('');
}

/** Màu avatar ổn định theo tên (hash đơn giản). */
export function avatarColor(seed: string): string {
  const palette = ['#4353e8', '#12b886', '#f59f00', '#e8590c', '#c026d3', '#0c8599'];
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) % 100000;
  }
  return palette[hash % palette.length]!;
}

/**
 * % hoàn thành suy ra từ task con trực tiếp (API không có field progress).
 * Trả null khi task không có task con.
 */
export function derivedProgress(
  subtasks: { statusId: string }[] | undefined,
  isDone: (statusId: string) => boolean,
): number | null {
  if (!subtasks || subtasks.length === 0) {
    return null;
  }
  const done = subtasks.filter((sub) => isDone(sub.statusId)).length;
  return Math.round((done / subtasks.length) * 100);
}
