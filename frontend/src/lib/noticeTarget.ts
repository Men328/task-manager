import { normalizeTargetType } from '../types';

/** Trang cần mở khi người dùng bấm vào một notice. */
export function noticeTargetPath(targetType: string): string {
  switch (normalizeTargetType(targetType)) {
    case 'schedule':
      return '/calendar';
    case 'event':
      return '/events';
    case 'backlog':
      return '/backlog';
    case 'friend':
      return '/friends';
    case 'message':
      return '/messages';
    default:
      return '/';
  }
}
