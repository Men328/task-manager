import { describe, expect, it } from 'vitest';

import { foldText, highlightParts } from './searchText';

describe('foldText', () => {
  it('drops accents, maps đ to d and lowercases', () => {
    expect(foldText('Báo Cáo ĐỒ ÁN')).toBe('bao cao do an');
  });

  it('treats decomposed input the same as precomposed', () => {
    expect(foldText('Báo')).toBe(foldText('Báo'));
  });
});

describe('highlightParts', () => {
  it('highlights the accented original for an unaccented query', () => {
    expect(highlightParts('Nộp báo cáo tuần', 'bao cao')).toEqual([
      { text: 'Nộp ', match: false },
      { text: 'báo cáo', match: true },
      { text: ' tuần', match: false },
    ]);
  });

  it('marks every occurrence and handles đ', () => {
    expect(highlightParts('Đồ án và đồ án', 'do an')).toEqual([
      { text: 'Đồ án', match: true },
      { text: ' và ', match: false },
      { text: 'đồ án', match: true },
    ]);
  });

  it('returns the text unchanged when nothing matches', () => {
    expect(highlightParts('Họp nhóm', 'xyz')).toEqual([{ text: 'Họp nhóm', match: false }]);
    expect(highlightParts('Họp nhóm', '  ')).toEqual([{ text: 'Họp nhóm', match: false }]);
  });
});
