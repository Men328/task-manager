import { describe, expect, it } from 'vitest';

import { matchSlashCommand } from './slashCommands';

describe('matchSlashCommand', () => {
  it('chỉ gõ "/" là hiện lệnh', () => {
    expect(matchSlashCommand('/')?.commands.map((c) => c.id)).toEqual(['task']);
  });

  it('khớp theo tiền tố, cả tên tiếng Việt', () => {
    expect(matchSlashCommand('/ta')?.commands[0].id).toBe('task');
    expect(matchSlashCommand('/gi')?.commands[0].id).toBe('task');
    expect(matchSlashCommand('/TASK')?.commands[0].id).toBe('task');
  });

  it('lấy phần sau tên lệnh làm tiêu đề', () => {
    expect(matchSlashCommand('/task  Mua sữa cho mẹ ')).toEqual(
      expect.objectContaining({ argument: 'Mua sữa cho mẹ' }),
    );
  });

  it('không phải lệnh', () => {
    expect(matchSlashCommand('hello /task')).toBeNull();
    expect(matchSlashCommand('/xyz')).toBeNull();
    expect(matchSlashCommand('/ta abc')).toBeNull();
    expect(matchSlashCommand('/task\nabc')).toBeNull();
  });
});
