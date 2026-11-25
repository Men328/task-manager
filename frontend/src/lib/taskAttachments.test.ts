import { describe, expect, it, vi } from 'vitest';

import {
  isRecentTask,
  mergeFiles,
  notifyTaskAttachmentsChanged,
  subscribeTaskAttachments,
} from './taskAttachments';

const file = (name: string, size = 1, lastModified = 1) => ({ name, size, lastModified });

describe('mergeFiles', () => {
  it('gộp tệp mới, bỏ tệp trùng và giữ thứ tự', () => {
    const merged = mergeFiles([file('a.pdf')], [file('b.png'), file('a.pdf'), file('c.txt')], 10);
    expect(merged.map((item) => item.name)).toEqual(['a.pdf', 'b.png', 'c.txt']);
  });

  it('coi cùng tên nhưng khác kích thước là tệp khác', () => {
    expect(mergeFiles([file('a.pdf', 1)], [file('a.pdf', 2)], 10)).toHaveLength(2);
  });

  it('dừng khi đủ giới hạn', () => {
    expect(mergeFiles([file('a')], [file('b'), file('c'), file('d')], 3).map((item) => item.name)).toEqual(['a', 'b', 'c']);
  });
});

describe('task attachment bus', () => {
  it('chỉ báo cho người nghe của đúng task', () => {
    const first = vi.fn();
    const other = vi.fn();
    const stopFirst = subscribeTaskAttachments('t1', first);
    const stopOther = subscribeTaskAttachments('t2', other);

    notifyTaskAttachmentsChanged('t1');
    expect(first).toHaveBeenCalledTimes(1);
    expect(other).not.toHaveBeenCalled();

    stopFirst();
    notifyTaskAttachmentsChanged('t1');
    expect(first).toHaveBeenCalledTimes(1);
    stopOther();
  });
});

describe('isRecentTask', () => {
  it('nhận tin trong vòng 60 giây', () => {
    const now = Date.parse('2026-11-24T10:01:00Z');
    expect(isRecentTask('2026-11-24T10:00:30Z', now)).toBe(true);
    expect(isRecentTask('2026-11-24T09:59:00Z', now)).toBe(false);
    expect(isRecentTask('', now)).toBe(false);
    expect(isRecentTask('không phải ngày', now)).toBe(false);
  });
});
