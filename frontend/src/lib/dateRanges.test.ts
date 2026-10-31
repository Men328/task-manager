import { describe, expect, it } from 'vitest';

import { addDays, agendaRanges, calendarDaysBetween, isLocalMidnight, localDayKey, startOfDay } from './dateRanges';

// Dùng constructor theo giờ địa phương để test không phụ thuộc múi giờ của máy chạy.
describe('agendaRanges', () => {
  it('uses local midnight boundaries for today, tomorrow and the 7-day window', () => {
    const { today0, tomorrow0, week0 } = agendaRanges(new Date(2026, 9, 31, 23, 30));
    expect(today0).toEqual(new Date(2026, 9, 31));
    expect(tomorrow0).toEqual(new Date(2026, 10, 1));
    expect(week0).toEqual(new Date(2026, 10, 8));
  });

  it('keeps a date-only due date (local 0h) in today, not overdue', () => {
    const { today0, tomorrow0 } = agendaRanges(new Date(2026, 9, 31, 0, 1));
    const dueToday = new Date(2026, 9, 31);
    expect(dueToday >= today0 && dueToday < tomorrow0).toBe(true);
  });

  it('crosses month and year ends', () => {
    expect(agendaRanges(new Date(2026, 11, 31, 12)).tomorrow0).toEqual(new Date(2027, 0, 1));
    expect(addDays(new Date(2026, 1, 28), 1)).toEqual(new Date(2026, 2, 1));
  });
});

describe('date helpers', () => {
  it('startOfDay drops the time part', () => {
    expect(startOfDay(new Date(2026, 4, 2, 17, 45, 12))).toEqual(new Date(2026, 4, 2));
  });

  it('localDayKey pads month and day', () => {
    expect(localDayKey(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05');
  });

  it('calendarDaysBetween counts calendar days regardless of time', () => {
    expect(calendarDaysBetween(new Date(2026, 9, 26, 23, 0), new Date(2026, 9, 31, 0, 30))).toBe(5);
    expect(calendarDaysBetween(new Date(2026, 9, 31, 8), new Date(2026, 9, 31, 22))).toBe(0);
  });

  it('isLocalMidnight detects date-only values', () => {
    expect(isLocalMidnight(new Date(2026, 9, 31))).toBe(true);
    expect(isLocalMidnight(new Date(2026, 9, 31, 15, 30))).toBe(false);
  });
});
