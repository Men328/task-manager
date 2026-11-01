/**
 * Parser cho ô "thêm nhanh": một dòng như `Nộp báo cáo mai 9h !cao #đồ-án` ->
 * tiêu đề + hạn + ưu tiên + project. Thuần (không đụng DOM/API) để test được.
 *
 * Quy ước:
 * - Chỉ nhận mẫu ở ranh giới từ; số trần ("chương 2") không bao giờ là ngày.
 * - Mỗi loại (ngày, giờ, ưu tiên, project) chỉ lấy lần xuất hiện CUỐI; lần trước giữ trong tiêu đề.
 * - Chỉ có ngày -> 0h giờ địa phương (khớp form task). Chỉ có giờ -> hôm nay, đã qua thì ngày mai.
 * - Bọc trong "..." để giữ nguyên văn.
 */

import { RRULE_DAYS } from '../recurrence';
import { foldText } from '../searchText';
import type { Project, TaskPriority } from '../../types';

export type QuickAddKind = 'date' | 'time' | 'priority' | 'project' | 'repeat';

export interface QuickAddToken {
  kind: QuickAddKind;
  start: number;
  end: number;
  text: string;
}

export type QuickAddWarning =
  | { kind: 'projectNotFound'; value: string }
  | { kind: 'projectAmbiguous'; value: string };

export interface QuickAddResult {
  title: string;
  dueAt?: Date;
  /** true nếu người dùng ghi giờ (hạn không phải 0h "chỉ có ngày"). */
  hasTime: boolean;
  priority?: TaskPriority;
  projectId?: string;
  /** Quy tắc lặp RRULE khi dòng nhập có "mỗi ngày", "hằng tuần", "every monday"… */
  recurrenceRule?: string;
  /** Token đã dùng (vị trí trong chuỗi gốc) để hiện chip. */
  tokens: QuickAddToken[];
  warnings: QuickAddWarning[];
}

export interface QuickAddOptions {
  now?: Date;
  projects?: Pick<Project, 'id' | 'name'>[];
  /** Loại token người dùng đã bỏ (bấm × trên chip): giữ chữ đó trong tiêu đề. */
  ignore?: ReadonlySet<QuickAddKind>;
}

type Resolver = (now: Date) => Date | null;
type TimeResolver = () => { hour: number; minute: number } | null;

interface Match {
  kind: QuickAddKind;
  start: number;
  end: number;
  text: string;
  date?: Resolver;
  time?: TimeResolver;
  priority?: TaskPriority;
  projectName?: string;
  /** first: lần đầu khi không ghi ngày; align: dời ngày người dùng ghi tới ngày đầu tiên khớp quy tắc. */
  repeat?: { rule: string; first: Resolver; align?: Resolver };
}

// Token phải đứng riêng: trước là đầu chuỗi/khoảng trắng/ngoặc mở, sau là cuối chuỗi/khoảng trắng/dấu câu.
// (Không dùng ranh giới chữ đơn thuần: "a@b.cn" không được hiểu "cn" là Chủ nhật.)
const B = '(?<=^|[\\s(\\[,;:])';
const E = '(?=$|[\\s)\\],.;:!?])';

const startOfDay = (date: Date) => new Date(date.getFullYear(), date.getMonth(), date.getDate());
const addDays = (date: Date, days: number) => new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);

/** Thứ sắp tới (0 = CN … 6 = T7); trùng hôm nay -> hôm nay. */
const nextWeekday = (now: Date, weekday: number) => addDays(startOfDay(now), (weekday - now.getDay() + 7) % 7);

/** Thứ Hai của tuần sau (tuần bắt đầu từ thứ Hai). */
const mondayNextWeek = (now: Date) => addDays(startOfDay(now), 7 - ((now.getDay() + 6) % 7));

const VI_WEEKDAY_WORD: Record<string, number> = { hai: 1, ba: 2, 'tư': 3, 'năm': 4, 'sáu': 5, 'bảy': 6 };
const EN_WEEKDAY: [RegExp, number][] = [
  [/^sun/, 0],
  [/^mon/, 1],
  [/^tue/, 2],
  [/^wed/, 3],
  [/^thu/, 4],
  [/^fri/, 5],
  [/^sat/, 6],
];

const VI_WEEKDAY = '(?:thứ\\s*([2-7])|thứ\\s+(hai|ba|tư|năm|sáu|bảy)|t([2-7])|(chủ\\s*nhật|cn))';
const EN_WEEKDAY_RE =
  '(monday|mon|tuesday|tues|tue|wednesday|wed|thursday|thurs|thur|thu|friday|fri|saturday|sat|sunday|sun)';

/** Thứ trong tuần (0–6) từ các nhóm của VI_WEEKDAY. */
function viWeekday(digit?: string, word?: string, short?: string, sunday?: string): number {
  if (sunday) {
    return 0;
  }
  if (word) {
    return VI_WEEKDAY_WORD[word.toLowerCase()] ?? 1;
  }
  return Number(digit ?? short) - 1;
}

function enWeekday(value: string): number {
  const lower = value.toLowerCase();
  return EN_WEEKDAY.find(([re]) => re.test(lower))?.[1] ?? 1;
}

const PRIORITY: Record<string, TaskPriority> = {
  'gấp': 'TASK_PRIORITY_URGENT',
  '1': 'TASK_PRIORITY_URGENT',
  urgent: 'TASK_PRIORITY_URGENT',
  cao: 'TASK_PRIORITY_HIGH',
  '2': 'TASK_PRIORITY_HIGH',
  high: 'TASK_PRIORITY_HIGH',
  tb: 'TASK_PRIORITY_MEDIUM',
  '3': 'TASK_PRIORITY_MEDIUM',
  medium: 'TASK_PRIORITY_MEDIUM',
  'thấp': 'TASK_PRIORITY_LOW',
  '4': 'TASK_PRIORITY_LOW',
  low: 'TASK_PRIORITY_LOW',
};

function validTime(hour: number, minute: number) {
  return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 ? { hour, minute } : null;
}

interface Pattern {
  kind: QuickAddKind;
  re: RegExp;
  build: (m: RegExpExecArray) => Omit<Match, 'kind' | 'start' | 'end' | 'text'> | null;
}

const today = (now: Date) => startOfDay(now);
const EVERY_VI = '(?:mỗi|hằng|hàng)';

/** Ngày gần nhất (kể cả hôm nay) có ngày-trong-tháng = day; bỏ qua tháng không có ngày đó. */
function nextMonthDay(now: Date, day: number): Date | null {
  for (let offset = 0; offset < 13; offset += 1) {
    const date = new Date(now.getFullYear(), now.getMonth() + offset, day);
    if (date.getDate() === day && date >= startOfDay(now)) {
      return date;
    }
  }
  return null;
}

const UNIT: Record<string, string> = {
  'ngày': 'DAILY',
  day: 'DAILY',
  days: 'DAILY',
  'tuần': 'WEEKLY',
  week: 'WEEKLY',
  weeks: 'WEEKLY',
  'tháng': 'MONTHLY',
  month: 'MONTHLY',
  months: 'MONTHLY',
};

// Thứ tự = độ ưu tiên khi các mẫu chồng nhau (mẫu dài/cụ thể trước).
const PATTERNS: Pattern[] = [
  {
    kind: 'repeat',
    re: new RegExp(`${B}mỗi\\s+ngày\\s+(\\d{1,2})${E}`, 'giu'),
    build: (m) => {
      const day = Number(m[1]);
      return day >= 1 && day <= 31
        ? {
            repeat: {
              rule: `FREQ=MONTHLY;BYMONTHDAY=${day}`,
              first: (now) => nextMonthDay(now, day),
              align: (date) => nextMonthDay(date, day),
            },
          }
        : null;
    },
  },
  {
    kind: 'repeat',
    re: new RegExp(`${B}(?:mỗi|every)\\s+(\\d{1,3})\\s+(ngày|tuần|tháng|days?|weeks?|months?)${E}`, 'giu'),
    build: (m) => {
      const interval = Number(m[1]);
      const unit = UNIT[m[2]!.toLowerCase()];
      if (!unit || interval < 1 || interval > 365) {
        return null;
      }
      return { repeat: { rule: interval === 1 ? `FREQ=${unit}` : `FREQ=${unit};INTERVAL=${interval}`, first: today } };
    },
  },
  {
    kind: 'repeat',
    re: new RegExp(`${B}(?:mỗi|every)\\s+${VI_WEEKDAY}${E}`, 'giu'),
    build: (m) => {
      const weekday = viWeekday(m[1], m[2], m[3], m[4]);
      return {
        repeat: {
          rule: `FREQ=WEEKLY;BYDAY=${RRULE_DAYS[weekday]}`,
          first: (now) => nextWeekday(now, weekday),
          align: (date) => nextWeekday(date, weekday),
        },
      };
    },
  },
  {
    kind: 'repeat',
    re: new RegExp(`${B}every\\s+${EN_WEEKDAY_RE}${E}`, 'giu'),
    build: (m) => {
      const weekday = enWeekday(m[1]!);
      return {
        repeat: {
          rule: `FREQ=WEEKLY;BYDAY=${RRULE_DAYS[weekday]}`,
          first: (now) => nextWeekday(now, weekday),
          align: (date) => nextWeekday(date, weekday),
        },
      };
    },
  },
  {
    kind: 'repeat',
    re: new RegExp(
      `${B}(${EVERY_VI}\\s+ngày|every\\s+day|daily|${EVERY_VI}\\s+tuần|every\\s+week|weekly|${EVERY_VI}\\s+tháng|every\\s+month|monthly|${EVERY_VI}\\s+năm|every\\s+year|yearly)${E}`,
      'giu',
    ),
    build: (m) => {
      const word = m[1]!.toLowerCase();
      const freq = /ngày|day|daily/.test(word)
        ? 'DAILY'
        : /tuần|week/.test(word)
          ? 'WEEKLY'
          : /tháng|month/.test(word)
            ? 'MONTHLY'
            : 'YEARLY';
      return { repeat: { rule: `FREQ=${freq}`, first: today } };
    },
  },
  {
    kind: 'project',
    re: /(?<!\S)#(?:"([^"]+)"|([^\s#"]+))/gu,
    build: (m) => ({ projectName: (m[1] ?? m[2] ?? '').trim() }),
  },
  {
    kind: 'priority',
    re: new RegExp(`(?<!\\S)!(gấp|cao|tb|thấp|urgent|high|medium|low|[1-4])${E}`, 'giu'),
    build: (m) => ({ priority: PRIORITY[m[1]!.toLowerCase()] }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}${VI_WEEKDAY}\\s+tuần\\s+sau${E}`, 'giu'),
    build: (m) => ({ date: (now) => addDays(mondayNextWeek(now), (viWeekday(m[1], m[2], m[3], m[4]) + 6) % 7) }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}${EN_WEEKDAY_RE}\\s+next\\s+week${E}`, 'giu'),
    build: (m) => ({ date: (now) => addDays(mondayNextWeek(now), (enWeekday(m[1]!) + 6) % 7) }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}(?:tuần\\s+sau|next\\s+week)${E}`, 'giu'),
    build: () => ({ date: mondayNextWeek }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}${VI_WEEKDAY}${E}`, 'giu'),
    build: (m) => ({ date: (now) => nextWeekday(now, viWeekday(m[1], m[2], m[3], m[4])) }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}${EN_WEEKDAY_RE}${E}`, 'giu'),
    build: (m) => ({ date: (now) => nextWeekday(now, enWeekday(m[1]!)) }),
  },
  {
    kind: 'date',
    re: new RegExp(`${B}(hôm\\s+nay|nay|today|ngày\\s+mai|mai|tomorrow|tmr|ngày\\s+kia|ngày\\s+mốt|mốt)${E}`, 'giu'),
    build: (m) => {
      const word = m[1]!.toLowerCase().replace(/\s+/g, ' ');
      const offset = ['hôm nay', 'nay', 'today'].includes(word) ? 0 : ['ngày mai', 'mai', 'tomorrow', 'tmr'].includes(word) ? 1 : 2;
      return { date: (now) => addDays(startOfDay(now), offset) };
    },
  },
  {
    kind: 'date',
    re: new RegExp(`${B}(\\d{1,2})[/-](\\d{1,2})(?:[/-](\\d{4}))?${E}`, 'gu'),
    build: (m) => {
      const day = Number(m[1]);
      const month = Number(m[2]) - 1;
      const year = m[3] ? Number(m[3]) : null;
      const make = (y: number) => {
        const date = new Date(y, month, day);
        return date.getMonth() === month && date.getDate() === day ? date : null;
      };
      if (!make(year ?? 2000)) {
        return null;
      }
      return {
        date: (now) => {
          if (year !== null) {
            return make(year);
          }
          const thisYear = make(now.getFullYear());
          return thisYear && thisYear >= startOfDay(now) ? thisYear : make(now.getFullYear() + 1);
        },
      };
    },
  },
  {
    kind: 'time',
    re: new RegExp(`${B}(\\d{1,2})(?::(\\d{2}))?\\s*(am|pm)${E}`, 'giu'),
    build: (m) => {
      const raw = Number(m[1]);
      if (raw < 1 || raw > 12) {
        return null;
      }
      const hour = (raw % 12) + (m[3]!.toLowerCase() === 'pm' ? 12 : 0);
      const time = validTime(hour, Number(m[2] ?? 0));
      return time ? { time: () => time } : null;
    },
  },
  {
    kind: 'time',
    re: new RegExp(`${B}(\\d{1,2})\\s*(?:giờ|h|g)(?:\\s*(\\d{2}))?${E}`, 'giu'),
    build: (m) => {
      const time = validTime(Number(m[1]), Number(m[2] ?? 0));
      return time ? { time: () => time } : null;
    },
  },
  {
    kind: 'time',
    re: new RegExp(`${B}(\\d{1,2}):(\\d{2})${E}`, 'gu'),
    build: (m) => {
      const time = validTime(Number(m[1]), Number(m[2]));
      return time ? { time: () => time } : null;
    },
  },
];

const QUOTED = /(?<!#)"([^"]*)"/gu;

function overlaps(a: { start: number; end: number }, b: { start: number; end: number }) {
  return a.start < b.end && b.start < a.end;
}

/** Chuẩn hoá tên project để so: bỏ dấu, chữ thường, `-`/`_` = dấu cách. */
const projectKey = (value: string) => foldText(value).replace(/[-_]+/g, ' ').replace(/\s+/g, ' ').trim();

function resolveProject(
  name: string,
  projects: Pick<Project, 'id' | 'name'>[],
): { id?: string; warning?: QuickAddWarning } {
  const key = projectKey(name);
  const exact = projects.filter((project) => projectKey(project.name) === key);
  if (exact.length === 1) {
    return { id: exact[0]!.id };
  }
  const prefix = projects.filter((project) => projectKey(project.name).startsWith(key));
  if (exact.length === 0 && prefix.length === 1) {
    return { id: prefix[0]!.id };
  }
  return { warning: { kind: exact.length + prefix.length === 0 ? 'projectNotFound' : 'projectAmbiguous', value: name } };
}

export function parseQuickAdd(input: string, options: QuickAddOptions = {}): QuickAddResult {
  const now = options.now ?? new Date();
  const ignore = options.ignore ?? new Set<QuickAddKind>();

  const protectedSpans: { start: number; end: number }[] = [];
  for (const m of input.matchAll(QUOTED)) {
    protectedSpans.push({ start: m.index, end: m.index + m[0].length });
  }

  const claimed: Match[] = [];
  for (const pattern of PATTERNS) {
    pattern.re.lastIndex = 0;
    for (const m of input.matchAll(pattern.re)) {
      const span = { start: m.index, end: m.index + m[0].length };
      if (protectedSpans.some((p) => overlaps(p, span)) || claimed.some((c) => overlaps(c, span))) {
        continue;
      }
      const built = pattern.build(m as RegExpExecArray);
      if (built) {
        claimed.push({ kind: pattern.kind, ...span, text: m[0], ...built });
      }
    }
  }

  // Mỗi loại chỉ dùng lần xuất hiện cuối cùng.
  const used = new Map<QuickAddKind, Match>();
  claimed
    .filter((match) => !ignore.has(match.kind) && !(match.kind === 'time' && ignore.has('date')))
    .forEach((match) => {
      const prev = used.get(match.kind);
      if (!prev || match.start > prev.start) {
        used.set(match.kind, match);
      }
    });

  const warnings: QuickAddWarning[] = [];
  let projectId: string | undefined;
  const projectMatch = used.get('project');
  if (projectMatch) {
    const resolved = resolveProject(projectMatch.projectName ?? '', options.projects ?? []);
    if (resolved.id) {
      projectId = resolved.id;
    } else {
      used.delete('project');
      if (resolved.warning) {
        warnings.push(resolved.warning);
      }
    }
  }

  let dueAt: Date | undefined;
  const dateMatch = used.get('date');
  const timeMatch = used.get('time');
  const repeatMatch = used.get('repeat');
  const repeatFirst = !dateMatch && repeatMatch?.repeat ? repeatMatch.repeat.first(now) : null;
  if (repeatMatch && !dateMatch && !repeatFirst) {
    used.delete('repeat');
  }
  let day = dateMatch?.date?.(now) ?? repeatFirst;
  const align = repeatMatch?.repeat?.align;
  if (dateMatch && day && align) {
    day = align(day) ?? day;
  }
  const time = timeMatch?.time?.() ?? null;
  if (dateMatch && !day) {
    used.delete('date');
  }
  if (repeatFirst && time && repeatMatch?.repeat?.rule.startsWith('FREQ=DAILY') && day) {
    const atTime = new Date(day.getFullYear(), day.getMonth(), day.getDate(), time.hour, time.minute);
    if (atTime <= now) {
      day = addDays(day, 1);
    }
  }
  if (day && time) {
    dueAt = new Date(day.getFullYear(), day.getMonth(), day.getDate(), time.hour, time.minute);
  } else if (day) {
    dueAt = day;
  } else if (time) {
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate(), time.hour, time.minute);
    dueAt = today > now ? today : addDays(today, 1);
    dueAt.setHours(time.hour, time.minute);
  }

  const tokens = [...used.values()]
    .sort((a, b) => a.start - b.start)
    .map(({ kind, start, end, text }) => ({ kind, start, end, text }));

  // Tiêu đề: bỏ token đã dùng, bỏ dấu nháy của đoạn giữ nguyên văn, gộp khoảng trắng.
  let title = '';
  let cursor = 0;
  const cuts = [
    ...tokens.map((token) => ({ start: token.start, end: token.end, keep: '' })),
    ...protectedSpans.map((span) => ({ start: span.start, end: span.end, keep: input.slice(span.start + 1, span.end - 1) })),
  ].sort((a, b) => a.start - b.start);
  for (const cut of cuts) {
    title += input.slice(cursor, cut.start) + (cut.keep ? cut.keep : ' ');
    cursor = cut.end;
  }
  title += input.slice(cursor);
  title = title.replace(/\s+/g, ' ').trim();

  return {
    title,
    dueAt,
    hasTime: Boolean(time && dueAt),
    priority: used.get('priority')?.priority,
    projectId,
    recurrenceRule: used.get('repeat')?.repeat?.rule,
    tokens,
    warnings,
  };
}
