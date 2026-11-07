import { describe, expect, it } from 'vitest';

import { noticeTargetPath } from './noticeTarget';

describe('noticeTargetPath', () => {
  it.each([
    ['NOTICE_TARGET_TYPE_FRIEND', '/friends'],
    ['friend', '/friends'],
    ['NOTICE_TARGET_TYPE_EVENT', '/events'],
    ['NOTICE_TARGET_TYPE_SCHEDULE', '/calendar'],
    ['NOTICE_TARGET_TYPE_BACKLOG', '/backlog'],
    ['NOTICE_TARGET_TYPE_TASK', '/'],
    ['không rõ', '/'],
  ])('%s -> %s', (targetType, path) => {
    expect(noticeTargetPath(targetType)).toBe(path);
  });
});
