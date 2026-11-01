import { useEffect, useRef, useState } from 'react';

import { getErrorMessage } from '../../api/client';
import { listStatuses, listTasksPage } from '../../api/task';
import type { Task, TaskStatus } from '../../types';

export const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;
const RESULT_LIMIT = 20;

export type SearchState =
  | { kind: 'idle' }
  | { kind: 'tooShort' }
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'done'; tasks: Task[] };

/** Tìm task của mọi project (debounce, bỏ response cũ) + danh sách status để hiển thị. */
export function useTaskSearch(profileId: string | null, query: string) {
  const [state, setState] = useState<SearchState>({ kind: 'idle' });
  const [statuses, setStatuses] = useState<Map<string, TaskStatus>>(new Map());
  const requestId = useRef(0);

  useEffect(() => {
    if (!profileId) {
      return;
    }
    let cancelled = false;
    listStatuses(profileId)
      .then((items) => {
        if (!cancelled) {
          setStatuses(new Map(items.map((status) => [status.id, status])));
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [profileId]);

  useEffect(() => {
    const trimmed = query.trim();
    const current = ++requestId.current;
    if (!profileId || trimmed.length === 0) {
      setState({ kind: 'idle' });
      return;
    }
    if ([...trimmed].length < MIN_QUERY_LENGTH) {
      setState({ kind: 'tooShort' });
      return;
    }
    setState({ kind: 'loading' });
    const timer = window.setTimeout(() => {
      listTasksPage({ profileId, query: trimmed, pageSize: RESULT_LIMIT })
        .then((page) => {
          if (current === requestId.current) {
            setState({ kind: 'done', tasks: page.tasks });
          }
        })
        .catch((cause) => {
          if (current === requestId.current) {
            setState({ kind: 'error', message: getErrorMessage(cause) });
          }
        });
    }, DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [profileId, query]);

  return { state, statuses };
}
