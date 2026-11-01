/** So khớp chuỗi tìm kiếm phía UI theo cùng quy tắc với backend: bỏ dấu, `đ` -> `d`, không phân biệt hoa thường. */

function foldChar(char: string): string {
  return char
    .normalize('NFD')
    .replace(/\p{Mn}/gu, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toLowerCase();
}

/** Chuỗi đã bỏ dấu + chữ thường. */
export function foldText(value: string): string {
  return [...value.normalize('NFC')].map(foldChar).join('');
}

export interface TextPart {
  text: string;
  match: boolean;
}

/**
 * Tách `text` thành các đoạn khớp / không khớp `query` (so sau khi bỏ dấu) để tô đậm,
 * giữ nguyên chữ có dấu gốc. Không khớp thì trả về một đoạn duy nhất.
 */
export function highlightParts(text: string, query: string): TextPart[] {
  const source = [...text.normalize('NFC')];
  const needle = foldText(query.trim());
  if (!needle) {
    return [{ text, match: false }];
  }

  // folded[i] là phần đã bỏ dấu của ký tự gốc source[i]; owner map mỗi ký tự folded về index gốc.
  let folded = '';
  const owner: number[] = [];
  source.forEach((char, index) => {
    const part = foldChar(char);
    folded += part;
    for (let i = 0; i < part.length; i += 1) {
      owner.push(index);
    }
  });

  const parts: TextPart[] = [];
  let cursor = 0;
  let from = folded.indexOf(needle);
  while (from !== -1) {
    const start = owner[from]!;
    const end = owner[from + needle.length - 1]! + 1;
    if (start > cursor) {
      parts.push({ text: source.slice(cursor, start).join(''), match: false });
    }
    parts.push({ text: source.slice(start, end).join(''), match: true });
    cursor = end;
    from = folded.indexOf(needle, from + needle.length);
  }
  if (cursor < source.length) {
    parts.push({ text: source.slice(cursor).join(''), match: false });
  }
  return parts;
}
