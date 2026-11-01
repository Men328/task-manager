import { describe, expect, it } from 'vitest';

import { presetOf, reminderFor, toLocalInput } from './reminders';

describe('reminderFor', () => {
  it('builds 9:00 on the due day and the day before, in local time', () => {
    expect(reminderFor('dueMorning', '2026-11-05', '')).toEqual(new Date(2026, 10, 5, 9, 0));
    expect(reminderFor('dayBefore', '2026-11-05', '')).toEqual(new Date(2026, 10, 4, 9, 0));
    expect(reminderFor('dayBefore', new Date(2026, 11, 1), '')).toEqual(new Date(2026, 10, 30, 9, 0));
  });

  it('accepts the ISO due date the API returns (local day of that instant)', () => {
    const due = new Date(2026, 10, 2, 9, 0).toISOString();
    expect(reminderFor('dueMorning', due, '')).toEqual(new Date(2026, 10, 2, 9, 0));
    expect(reminderFor('dayBefore', due, '')).toEqual(new Date(2026, 10, 1, 9, 0));
    expect(presetOf(new Date(2026, 10, 2, 9, 0), due)).toBe('dueMorning');
  });

  it('needs a due date for due-based presets and nothing for none', () => {
    expect(reminderFor('dueMorning', null, '')).toBeNull();
    expect(reminderFor('none', '2026-11-05', '')).toBeNull();
  });

  it('reads custom datetime-local values and rejects empty ones', () => {
    expect(reminderFor('custom', null, '2026-11-02T14:30')).toEqual(new Date(2026, 10, 2, 14, 30));
    expect(reminderFor('custom', null, '')).toBeNull();
  });
});

describe('presetOf', () => {
  it('recognises presets and falls back to custom', () => {
    expect(presetOf(null, '2026-11-05')).toBe('none');
    expect(presetOf(new Date(2026, 10, 5, 9, 0), '2026-11-05')).toBe('dueMorning');
    expect(presetOf(new Date(2026, 10, 4, 9, 0), '2026-11-05')).toBe('dayBefore');
    expect(presetOf(new Date(2026, 10, 4, 10, 0), '2026-11-05')).toBe('custom');
  });

  it('round-trips custom values through the input format', () => {
    expect(toLocalInput(new Date(2026, 0, 2, 3, 4))).toBe('2026-01-02T03:04');
  });
});
