import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { getErrorMessage } from '../../api/client';
import { changeTaskStatus, getTask, listAllTasks, listStatuses, listTransitions } from '../../api/task';
import { agendaRanges } from '../../lib/dateRanges';
import type { StatusTransition, Task, TaskStatus } from '../../types';

export type AgendaTab = 'today' | 'upcoming' | 'overdue';

interface AgendaData {
  overdue: Task[];
  today: Task[];
  upcoming: Task[];
  statuses: TaskStatus[];
  transitions: StatusTransition[];
}

const EMPTY: AgendaData = { overdue: [], today: [], upcoming: [], statuses: [], transitions: [] };

/**
 * Tải task có hạn của mọi project cho 3 mốc (quá hạn / hôm nay / 7 ngày tới) cùng lúc
 * để tab hiện được số lượng. Chỉ task đang mở (status không thuộc done/cancelled).
 */
export function useAgenda(profileId: string | null) {
  const [data, setData] = useState<AgendaData>(EMPTY);
  const [parentTitles, setParentTitles] = useState<Map<string, string>>(new Map());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const requestId = useRef(0);

  const load = useCallback(async () => {
    if (!profileId) {
      setData(EMPTY);
      return;
    }
    const current = ++requestId.current;
    setLoading(true);
    setError(null);
    const { today0, tomorrow0, week0 } = agendaRanges();
    const base = { profileId, openOnly: true, sortField: 'TASK_SORT_FIELD_DUE_AT' as const };
    try {
      const [overdue, today, upcoming, statuses, transitions] = await Promise.all([
        listAllTasks({ ...base, dueTo: today0.toISOString() }),
        listAllTasks({ ...base, dueFrom: today0.toISOString(), dueTo: tomorrow0.toISOString() }),
        listAllTasks({ ...base, dueFrom: tomorrow0.toISOString(), dueTo: week0.toISOString() }),
        listStatuses(profileId),
        listTransitions(profileId),
      ]);
      // Bỏ response cũ nếu đã có lần tải mới hơn (đổi profile, bấm tải lại liên tục).
      if (current === requestId.current) {
        setData({ overdue, today, upcoming, statuses, transitions });
      }
    } catch (cause) {
      if (current === requestId.current) {
        setError(getErrorMessage(cause));
      }
    } finally {
      if (current === requestId.current) {
        setLoading(false);
      }
    }
  }, [profileId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const onFocus = () => void load();
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [load]);

  // Tiêu đề task cha cho dòng "↳ trong: ...": tải những cha chưa có trong danh sách.
  useEffect(() => {
    const all = [...data.overdue, ...data.today, ...data.upcoming];
    const known = new Map(all.map((task) => [task.id, task.title]));
    const missing = [
      ...new Set(
        all
          .map((task) => task.parentTaskId)
          .filter((id): id is string => Boolean(id) && !known.has(id as string)),
      ),
    ];
    let cancelled = false;
    void Promise.all(missing.map((id) => getTask(id, false).catch(() => null))).then((parents) => {
      if (cancelled) {
        return;
      }
      parents.forEach((parent) => {
        if (parent) {
          known.set(parent.id, parent.title);
        }
      });
      setParentTitles(known);
    });
    return () => {
      cancelled = true;
    };
  }, [data]);

  // Backend chỉ cho đổi status khi có transition đang bật (whitelist), nên UI tính trước
  // để không hiện nút chắc chắn thất bại.
  const allowed = useMemo(
    () =>
      new Set(
        data.transitions
          .filter((transition) => transition.isActive !== false)
          .map((transition) => `${transition.fromStatusId}>${transition.toStatusId}`),
      ),
    [data.transitions],
  );

  const doneStatuses = useMemo(
    () =>
      data.statuses
        .filter((status) => status.category === 'TASK_STATUS_CATEGORY_DONE' && !status.isArchived)
        .sort((a, b) => (a.position ?? 0) - (b.position ?? 0)),
    [data.statuses],
  );

  /** Status nhóm done đầu tiên chuyển thẳng tới được từ status hiện tại của task; null nếu không có. */
  const doneTargetFor = useCallback(
    (task: Task) => doneStatuses.find((status) => allowed.has(`${task.statusId}>${status.id}`)) ?? null,
    [allowed, doneStatuses],
  );

  const canTransition = useCallback(
    (fromStatusId: string, toStatusId: string) => allowed.has(`${fromStatusId}>${toStatusId}`),
    [allowed],
  );

  const statusName = useCallback(
    (statusId: string) => data.statuses.find((status) => status.id === statusId)?.name ?? '',
    [data.statuses],
  );

  /** Đánh dấu xong; trả status đích để có thể hoàn tác. */
  const complete = useCallback(
    async (task: Task) => {
      const target = doneTargetFor(task);
      if (!target) {
        throw new Error('no done status reachable');
      }
      const result = await changeTaskStatus(task.id, target.id);
      return { target, result };
    },
    [doneTargetFor],
  );

  const restoreStatus = useCallback(async (task: Task, statusId: string) => {
    await changeTaskStatus(task.id, statusId);
  }, []);

  return {
    ...data,
    parentTitles,
    loading,
    error,
    reload: load,
    hasDoneStatus: doneStatuses.length > 0,
    doneTargetFor,
    canTransition,
    statusName,
    complete,
    restoreStatus,
  };
}
