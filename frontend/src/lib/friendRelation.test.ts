import { describe, expect, it } from 'vitest';

import { relationAction } from './friendRelation';

describe('relationAction', () => {
  it.each([
    ['none', 'add'],
    ['incoming', 'accept'],
    ['outgoing', 'sent'],
    ['friend', 'friend'],
    ['self', null],
  ] as const)('%s -> %s', (relation, action) => {
    expect(relationAction(relation)).toBe(action);
  });
});
