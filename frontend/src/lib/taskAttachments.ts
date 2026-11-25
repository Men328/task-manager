/**
 * Tệp đính kèm khi giao task trong chat.
 *
 * Task được tạo ở phía người nhận nên chỉ có id sau khi gửi xong; tệp được tải lên
 * ngay sau đó và gắn vào task (`owner_type = task`). Một "bus" nhỏ báo cho mọi nơi đang
 * hiển thị tệp của task đó (thẻ task trong chat, khung chi tiết task) tải lại danh sách.
 */

type Listener = () => void;

const listeners = new Map<string, Set<Listener>>();

/** Đăng ký nhận báo khi tệp của task đổi; trả hàm huỷ đăng ký. */
export function subscribeTaskAttachments(taskId: string, listener: Listener): () => void {
  let set = listeners.get(taskId);
  if (!set) {
    set = new Set();
    listeners.set(taskId, set);
  }
  set.add(listener);
  return () => {
    const current = listeners.get(taskId);
    if (!current) {
      return;
    }
    current.delete(listener);
    if (current.size === 0) {
      listeners.delete(taskId);
    }
  };
}

/** Báo tệp của task vừa đổi (vd. vừa tải lên khi giao task). */
export function notifyTaskAttachmentsChanged(taskId: string): void {
  listeners.get(taskId)?.forEach((listener) => listener());
}

/** Khoá nhận diện một tệp đã chọn, để không thêm trùng cùng một tệp. */
export function fileKey(file: Pick<File, 'name' | 'size' | 'lastModified'>): string {
  return `${file.name}:${file.size}:${file.lastModified}`;
}

/** Gộp tệp mới chọn vào danh sách, bỏ tệp trùng, giữ thứ tự chọn. */
export function mergeFiles<T extends Pick<File, 'name' | 'size' | 'lastModified'>>(current: T[], added: T[], limit: number): T[] {
  const seen = new Set(current.map(fileKey));
  const out = [...current];
  for (const file of added) {
    if (out.length >= limit) {
      break;
    }
    const key = fileKey(file);
    if (!seen.has(key)) {
      seen.add(key);
      out.push(file);
    }
  }
  return out;
}

/**
 * Tin nhắn task vừa tạo thì người gửi có thể còn đang tải tệp lên: thẻ task thử tải lại
 * danh sách vài lần trong khoảng này (người nhận không nhận được báo từ bus của người gửi).
 */
export const RECENT_TASK_WINDOW_MS = 60_000;
export const RECENT_TASK_RETRY_DELAYS_MS = [2_000, 5_000, 10_000, 20_000];

export function isRecentTask(createdAt: string | null | undefined, now: number = Date.now()): boolean {
  if (!createdAt) {
    return false;
  }
  const created = new Date(createdAt).getTime();
  return !Number.isNaN(created) && now - created < RECENT_TASK_WINDOW_MS;
}
