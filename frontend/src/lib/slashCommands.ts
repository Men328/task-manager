export type SlashCommandId = 'task';

export interface SlashCommand {
  id: SlashCommandId;
  /** Tên gõ sau dấu `/`; khớp theo tiền tố. */
  names: string[];
}

export const SLASH_COMMANDS: SlashCommand[] = [{ id: 'task', names: ['task', 'giao'] }];

export interface SlashMatch {
  commands: SlashCommand[];
  /** Phần chữ sau tên lệnh, dùng làm tiêu đề task điền sẵn. */
  argument: string;
}

/**
 * Ô chat đang gõ lệnh khi bắt đầu bằng `/` và chỉ có 1 dòng.
 * Trả về null khi không phải lệnh hoặc không lệnh nào khớp.
 */
export function matchSlashCommand(draft: string): SlashMatch | null {
  if (!draft.startsWith('/') || draft.includes('\n')) {
    return null;
  }
  const rest = draft.slice(1);
  const spaceAt = rest.search(/\s/);
  const name = (spaceAt === -1 ? rest : rest.slice(0, spaceAt)).toLowerCase();
  const argument = spaceAt === -1 ? '' : rest.slice(spaceAt + 1).trim();

  const commands = SLASH_COMMANDS.filter((command) =>
    command.names.some((candidate) => (spaceAt === -1 ? candidate.startsWith(name) : candidate === name)),
  );
  return commands.length > 0 ? { commands, argument } : null;
}
