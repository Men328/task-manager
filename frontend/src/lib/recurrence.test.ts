import { describe, expect, it } from 'vitest';

import { describeRule, presetOf, resolveRule, ruleFor, rulesEqual } from './recurrence';

// Thứ Sáu 06/11/2026.
const FRI = '2026-11-06';

const t = (key: string, options?: Record<string, unknown>) =>
  `${key}${options ? ` ${JSON.stringify(options)}` : ''}`;

describe('ruleFor', () => {
  it.each([
    ['daily', 'FREQ=DAILY'],
    ['weekdays', 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR'],
    ['weekly', 'FREQ=WEEKLY;BYDAY=FR'],
    ['monthly', 'FREQ=MONTHLY;BYMONTHDAY=6'],
    ['yearly', 'FREQ=YEARLY'],
  ] as const)('%s', (preset, rule) => {
    expect(ruleFor(preset, FRI)).toBe(rule);
  });

  it('builds custom intervals and clamps them', () => {
    expect(ruleFor('custom', FRI, { interval: 3, unit: 'WEEKLY' })).toBe('FREQ=WEEKLY;INTERVAL=3');
    expect(ruleFor('custom', FRI, { interval: 1, unit: 'MONTHLY' })).toBe('FREQ=MONTHLY');
    expect(ruleFor('custom', FRI, { interval: 9999, unit: 'DAILY' })).toBe('FREQ=DAILY;INTERVAL=365');
  });

  it('needs a due date and returns nothing for none', () => {
    expect(ruleFor('daily', null)).toBeNull();
    expect(ruleFor('none', FRI)).toBeNull();
  });

  it('uses the local day of an ISO due date', () => {
    expect(ruleFor('weekly', new Date(2026, 10, 6, 0, 0).toISOString())).toBe('FREQ=WEEKLY;BYDAY=FR');
  });
});

describe('presetOf', () => {
  it('maps known rules back to presets and the rest to custom', () => {
    expect(presetOf('FREQ=WEEKLY;BYDAY=FR', FRI).preset).toBe('weekly');
    expect(presetOf('FREQ=MONTHLY;BYMONTHDAY=6', FRI).preset).toBe('monthly');
    expect(presetOf('FREQ=WEEKLY;INTERVAL=2', FRI)).toEqual({ preset: 'custom', custom: { interval: 2, unit: 'WEEKLY' } });
    expect(presetOf('', FRI).preset).toBe('none');
  });
});

describe('describeRule', () => {
  it.each([
    ['FREQ=DAILY', 'repeat.describe.daily'],
    ['FREQ=DAILY;INTERVAL=3', 'repeat.describe.everyNDays {"count":3}'],
    ['FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR', 'repeat.describe.weekdays'],
    ['FREQ=WEEKLY;BYDAY=FR', 'repeat.describe.weeklyOn {"days":"Friday"}'],
    ['FREQ=WEEKLY;INTERVAL=2', 'repeat.describe.everyNWeeks {"count":2}'],
    ['FREQ=MONTHLY;BYMONTHDAY=5', 'repeat.describe.monthlyOn {"day":"5"}'],
    ['FREQ=YEARLY', 'repeat.describe.yearly'],
  ])('%s', (rule, label) => {
    expect(describeRule(rule, t, 'en-US')).toBe(label);
  });

  it('returns null without a rule', () => {
    expect(describeRule('', t, 'vi')).toBeNull();
  });
});

describe('rules the presets cannot express (review fix)', () => {
  it.each([
    'FREQ=DAILY;COUNT=3',
    'FREQ=WEEKLY;BYDAY=MO,WE',
    'FREQ=YEARLY;INTERVAL=2',
    'FREQ=MONTHLY;BYMONTHDAY=5',
    'FREQ=WEEKLY;UNTIL=20261231T000000Z',
  ])('%s is kept as-is', (rule) => {
    const { preset, custom } = presetOf(rule, FRI);
    expect(preset).toBe('current');
    expect(resolveRule(preset, FRI, custom, rule)).toBe(rule);
  });

  it('compares rules regardless of property order and INTERVAL=1', () => {
    expect(rulesEqual('FREQ=WEEKLY;INTERVAL=2;BYDAY=FR', 'BYDAY=FR;FREQ=WEEKLY;INTERVAL=2')).toBe(true);
    expect(rulesEqual('FREQ=DAILY;INTERVAL=1', 'FREQ=DAILY')).toBe(true);
    expect(rulesEqual('FREQ=DAILY;COUNT=2', 'FREQ=DAILY')).toBe(false);
    expect(presetOf('INTERVAL=3;FREQ=WEEKLY', FRI)).toEqual({ preset: 'custom', custom: { interval: 3, unit: 'WEEKLY' } });
  });

  it('describes COUNT and UNTIL', () => {
    expect(describeRule('FREQ=DAILY;COUNT=2', t, 'en-US')).toBe('repeat.describe.daily · repeat.describe.count {"count":2}');
    expect(describeRule('FREQ=DAILY;UNTIL=20261231T000000Z', t, 'en-US')).toBe(
      'repeat.describe.daily · repeat.describe.until {"date":"12/31/2026"}',
    );
  });
});
