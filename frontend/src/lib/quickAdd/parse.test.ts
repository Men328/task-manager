import { describe, expect, it } from 'vitest';

import { parseQuickAdd } from './parse';
import type { QuickAddKind } from './parse';

// Thứ Bảy 31/10/2026 10:00 và thứ Tư 04/11/2026 10:00 (giờ địa phương).
const SAT = new Date(2026, 9, 31, 10, 0);
const WED = new Date(2026, 10, 4, 10, 0);
const PROJECTS = [
  { id: 'p-doan', name: 'Đồ án tốt nghiệp' },
  { id: 'p-nha', name: 'Việc nhà' },
  { id: 'p-uat1', name: 'UAT-01' },
  { id: 'p-uat2', name: 'UAT-02' },
];

const d = (month: number, day: number, hour = 0, minute = 0, year = 2026) => new Date(year, month - 1, day, hour, minute);
const parse = (input: string, now = SAT, ignore?: QuickAddKind[]) =>
  parseQuickAdd(input, { now, projects: PROJECTS, ignore: ignore ? new Set(ignore) : undefined });

describe('dates', () => {
  it.each([
    ['Họp hôm nay', SAT, d(10, 31)],
    ['Họp nay', SAT, d(10, 31)],
    ['Họp today', SAT, d(10, 31)],
    ['Họp mai', SAT, d(11, 1)],
    ['Họp ngày mai', SAT, d(11, 1)],
    ['Họp tomorrow', SAT, d(11, 1)],
    ['Họp tmr', SAT, d(11, 1)],
    ['Họp mốt', SAT, d(11, 2)],
    ['Họp ngày kia', SAT, d(11, 2)],
    ['Họp thứ 6', WED, d(11, 6)],
    ['Họp t6', WED, d(11, 6)],
    ['Họp thứ sáu', WED, d(11, 6)],
    ['Họp Thứ Hai', WED, d(11, 9)],
    ['Họp chủ nhật', WED, d(11, 8)],
    ['Họp cn', WED, d(11, 8)],
    ['Họp friday', WED, d(11, 6)],
    ['Họp mon', WED, d(11, 9)],
    ['Họp thứ 4', WED, d(11, 4)],
    ['Họp tuần sau', WED, d(11, 9)],
    ['Họp next week', WED, d(11, 9)],
    ['Họp thứ 6 tuần sau', WED, d(11, 13)],
    ['Họp friday next week', WED, d(11, 13)],
    ['Họp tuần sau', SAT, d(11, 2)],
    ['Họp 15/11', SAT, d(11, 15)],
    ['Họp 15-11', SAT, d(11, 15)],
    ['Họp 20/3/2027', SAT, d(3, 20, 0, 0, 2027)],
    ['Họp 1/1', SAT, d(1, 1, 0, 0, 2027)],
    ['Họp 31/10', SAT, d(10, 31)],
  ])('%s', (input, now, due) => {
    const result = parse(input, now);
    expect(result.dueAt).toEqual(due);
    expect(result.title).toBe('Họp');
    expect(result.hasTime).toBe(false);
  });

  it('ignores impossible dates and keeps them in the title', () => {
    const result = parse('Kiểm kê 31/2');
    expect(result.dueAt).toBeUndefined();
    expect(result.title).toBe('Kiểm kê 31/2');
  });
});

describe('times', () => {
  it.each([
    ['Gọi mai 9h', d(11, 1, 9)],
    ['Gọi mai 9h30', d(11, 1, 9, 30)],
    ['Gọi mai 21h', d(11, 1, 21)],
    ['Gọi mai 9:30', d(11, 1, 9, 30)],
    ['Gọi mai 9 giờ', d(11, 1, 9)],
    ['Gọi mai 9g', d(11, 1, 9)],
    ['Gọi tomorrow 9am', d(11, 1, 9)],
    ['Gọi tomorrow 9:30pm', d(11, 1, 21, 30)],
    ['Gọi tomorrow 12am', d(11, 1, 0)],
    ['Gọi 15h', d(10, 31, 15)],
    ['Gọi 9h', d(11, 1, 9)],
  ])('%s', (input, due) => {
    const result = parse(input);
    expect(result.dueAt).toEqual(due);
    expect(result.hasTime).toBe(true);
    expect(result.title).toBe('Gọi');
  });

  it('rejects invalid hours', () => {
    expect(parse('Ca 25h').dueAt).toBeUndefined();
    expect(parse('Ca 13pm').dueAt).toBeUndefined();
  });
});

describe('priority and project', () => {
  it.each([
    ['!gấp', 'TASK_PRIORITY_URGENT'],
    ['!1', 'TASK_PRIORITY_URGENT'],
    ['!cao', 'TASK_PRIORITY_HIGH'],
    ['!high', 'TASK_PRIORITY_HIGH'],
    ['!tb', 'TASK_PRIORITY_MEDIUM'],
    ['!thấp', 'TASK_PRIORITY_LOW'],
    ['!4', 'TASK_PRIORITY_LOW'],
    ['!LOW', 'TASK_PRIORITY_LOW'],
  ])('priority %s', (token, priority) => {
    const result = parse(`Sửa lỗi ${token}`);
    expect(result.priority).toBe(priority);
    expect(result.title).toBe('Sửa lỗi');
  });

  it('matches projects ignoring accents, case and dashes', () => {
    expect(parse('Viết báo cáo #do-an-tot-nghiep').projectId).toBe('p-doan');
    expect(parse('Viết báo cáo #"Đồ án tốt nghiệp"').projectId).toBe('p-doan');
    expect(parse('Rửa bát #viec_nha').projectId).toBe('p-nha');
    expect(parse('Rửa bát #viec').projectId).toBe('p-nha');
    expect(parse('Test #uat-01').projectId).toBe('p-uat1');
  });

  it('warns and keeps the text when the project is unknown or ambiguous', () => {
    const missing = parse('Mua sữa #chợ');
    expect(missing.projectId).toBeUndefined();
    expect(missing.title).toBe('Mua sữa #chợ');
    expect(missing.warnings).toEqual([{ kind: 'projectNotFound', value: 'chợ' }]);

    const ambiguous = parse('Test #uat');
    expect(ambiguous.projectId).toBeUndefined();
    expect(ambiguous.warnings).toEqual([{ kind: 'projectAmbiguous', value: 'uat' }]);
  });
});

describe('false positives and edge cases', () => {
  it.each([
    'Đọc chương 2',
    'Mua 2 hộp sữa',
    'Phòng 302',
    'Version 1.2',
    'Email a@b.cn',
    'Học thứ tự ưu tiên',
  ])('%s has no due date', (input) => {
    const result = parse(input);
    expect(result.dueAt).toBeUndefined();
    expect(result.title).toBe(input);
  });

  it('uses only the last date and keeps earlier ones in the title', () => {
    const result = parse('Dời họp thứ 2 sang thứ 4', WED);
    expect(result.dueAt).toEqual(d(11, 4));
    expect(result.title).toBe('Dời họp thứ 2 sang');
  });

  it('keeps quoted text verbatim', () => {
    const result = parse('Đọc "Hẹn em ngày mai" !cao');
    expect(result.title).toBe('Đọc Hẹn em ngày mai');
    expect(result.dueAt).toBeUndefined();
    expect(result.priority).toBe('TASK_PRIORITY_HIGH');
  });

  it('parses everything together and reports tokens in order', () => {
    const input = 'Nộp báo cáo mai 9h !cao #do-an';
    const result = parse(input);
    expect(result).toMatchObject({
      title: 'Nộp báo cáo',
      dueAt: d(11, 1, 9),
      hasTime: true,
      priority: 'TASK_PRIORITY_HIGH',
      projectId: 'p-doan',
      warnings: [],
    });
    expect(result.tokens.map((token) => [token.kind, token.text])).toEqual([
      ['date', 'mai'],
      ['time', '9h'],
      ['priority', '!cao'],
      ['project', '#do-an'],
    ]);
    expect(result.tokens.map((token) => input.slice(token.start, token.end))).toEqual(['mai', '9h', '!cao', '#do-an']);
  });

  it('ignored kinds stay in the title (removing the due chip ignores date and time)', () => {
    const result = parse('Nộp báo cáo mai 9h !cao', SAT, ['date', 'priority']);
    expect(result.dueAt).toBeUndefined();
    expect(result.priority).toBeUndefined();
    expect(result.title).toBe('Nộp báo cáo mai 9h !cao');
  });

  it('returns an empty title when only tokens are typed', () => {
    expect(parse('mai !cao').title).toBe('');
  });
});
