import { describe, expect, it } from 'vitest';

import { projectColor } from './tokens';

describe('projectColor', () => {
  it('uses the project color when set', () => {
    expect(projectColor({ id: 'p1', color: '#123456' })).toBe('#123456');
  });

  it('gives colorless projects a stable fallback that is never the brand color', () => {
    const ids = Array.from({ length: 50 }, (_, i) => `project-${i}`);
    for (const id of ids) {
      const color = projectColor({ id, color: '' });
      expect(color).toBe(projectColor({ id, color: '' }));
      expect(color.toLowerCase()).not.toBe('#4353e8');
    }
  });
});
