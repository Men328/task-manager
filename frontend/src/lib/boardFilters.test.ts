import { describe, expect, it } from 'vitest';

import {
  activeFilterCount,
  matchesBoardFilters,
  parseBoardFilters,
  sortBoardTasks,
  writeBoardFilters,
} from './boardFilters';
import type { BoardFilters } from './boardFilters';
import type { Task } from '../types';

const NOW = new Date(2026, 9, 31, 10, 0);
const local = (day: number, hour = 0) => new Date(2026, 9, day, hour).toISOString();

function task(id: string, patch: Partial<Task> = {}): Task {
  return {
    id,
    profileId: 'u1',
    projectId: 'p1',
    statusId: 'open',
    title: id,
    priority: 'TASK_PRIORITY_MEDIUM',
    ...patch,
  };
}

const base: BoardFilters = parseBoardFilters(new URLSearchParams());
const ctx = { isClosed: (statusId: string) => statusId === 'done', now: NOW };

describe('parse/write board filters', () => {
  it('defaults when the URL has no params', () => {
    expect(base).toEqual({ due: 'all', priorities: [], hideDone: false, sort: { field: 'position', desc: false } });
    expect(activeFilterCount(base)).toBe(0);
  });

  it('round-trips through the query string and keeps unrelated params', () => {
    const filters: BoardFilters = {
      due: 'overdue',
      priorities: ['TASK_PRIORITY_URGENT', 'TASK_PRIORITY_HIGH'],
      hideDone: true,
      sort: { field: 'due', desc: true },
    };
    const params = writeBoardFilters(new URLSearchParams('view=table'), filters);
    expect(params.toString()).toBe('view=table&due=overdue&prio=high%2Curgent&hideDone=1&sort=due%3Adesc');
    expect(parseBoardFilters(params)).toEqual({ ...filters, priorities: ['TASK_PRIORITY_HIGH', 'TASK_PRIORITY_URGENT'] });
    expect(activeFilterCount(filters)).toBe(3);
  });

  it('drops default values and ignores junk', () => {
    expect(writeBoardFilters(new URLSearchParams('due=today&sort=due:asc'), base).toString()).toBe('');
    expect(parseBoardFilters(new URLSearchParams('due=soon&prio=HIGH,nope&sort=zzz:desc&hideDone=yes'))).toEqual({
      ...base,
      priorities: ['TASK_PRIORITY_HIGH'],
    });
  });
});

describe('matchesBoardFilters', () => {
  it('treats a date-only task due today (local 0h) as today, not overdue', () => {
    const dueToday = task('a', { dueAt: local(31) });
    expect(matchesBoardFilters(dueToday, { ...base, due: 'today' }, ctx)).toBe(true);
    expect(matchesBoardFilters(dueToday, { ...base, due: 'overdue' }, ctx)).toBe(false);
  });

  it('overdue excludes closed tasks and tasks without a due date', () => {
    expect(matchesBoardFilters(task('a', { dueAt: local(29) }), { ...base, due: 'overdue' }, ctx)).toBe(true);
    expect(matchesBoardFilters(task('b', { dueAt: local(29), statusId: 'done' }), { ...base, due: 'overdue' }, ctx)).toBe(false);
    expect(matchesBoardFilters(task('c'), { ...base, due: 'overdue' }, ctx)).toBe(false);
  });

  it('week covers today through the next 7 days', () => {
    const week = { ...base, due: 'week' as const };
    expect(matchesBoardFilters(task('a', { dueAt: local(31) }), week, ctx)).toBe(true);
    expect(matchesBoardFilters(task('b', { dueAt: new Date(2026, 10, 7, 23).toISOString() }), week, ctx)).toBe(true);
    expect(matchesBoardFilters(task('c', { dueAt: new Date(2026, 10, 8).toISOString() }), week, ctx)).toBe(false);
  });

  it('combines none, priority, hideDone and query', () => {
    expect(matchesBoardFilters(task('a'), { ...base, due: 'none' }, ctx)).toBe(true);
    expect(matchesBoardFilters(task('b', { dueAt: local(31) }), { ...base, due: 'none' }, ctx)).toBe(false);
    expect(matchesBoardFilters(task('c'), { ...base, priorities: ['TASK_PRIORITY_HIGH'] }, ctx)).toBe(false);
    expect(matchesBoardFilters(task('d', { statusId: 'done' }), { ...base, hideDone: true }, ctx)).toBe(false);
    expect(matchesBoardFilters(task('Nộp Báo cáo'), base, { ...ctx, query: 'báo cáo' })).toBe(true);
    expect(matchesBoardFilters(task('Nộp Báo cáo'), base, { ...ctx, query: 'họp' })).toBe(false);
  });
});

describe('sortBoardTasks', () => {
  const tasks = [
    task('a', { dueAt: local(30), priority: 'TASK_PRIORITY_LOW', position: 2 }),
    task('b', { priority: 'TASK_PRIORITY_URGENT', position: 1 }),
    task('c', { dueAt: local(29), priority: 'TASK_PRIORITY_HIGH', position: 0 }),
  ];
  const ids = (list: Task[]) => list.map((item) => item.id);

  it('sorts by position by default and does not mutate the input', () => {
    expect(ids(sortBoardTasks(tasks, { field: 'position', desc: false }))).toEqual(['c', 'b', 'a']);
    expect(ids(tasks)).toEqual(['a', 'b', 'c']);
  });

  it('keeps tasks without a due date last in both directions', () => {
    expect(ids(sortBoardTasks(tasks, { field: 'due', desc: false }))).toEqual(['c', 'a', 'b']);
    expect(ids(sortBoardTasks(tasks, { field: 'due', desc: true }))).toEqual(['a', 'c', 'b']);
  });

  it('sorts priority descending as urgent first', () => {
    expect(ids(sortBoardTasks(tasks, { field: 'priority', desc: true }))).toEqual(['b', 'c', 'a']);
  });
});
