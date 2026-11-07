import type { FriendRelation } from '../types';

export type RelationAction = 'add' | 'accept' | 'sent' | 'friend';

/** Nút hiển thị trên kết quả tìm kiếm; `null` = không có nút (chính mình). */
export function relationAction(relation: FriendRelation): RelationAction | null {
  switch (relation) {
    case 'none':
      return 'add';
    case 'incoming':
      return 'accept';
    case 'outgoing':
      return 'sent';
    case 'friend':
      return 'friend';
    default:
      return null;
  }
}
