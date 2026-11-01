import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

import { parseBoardFilters, writeBoardFilters } from '../../lib/boardFilters';
import type { BoardFilters } from '../../lib/boardFilters';

/** Bộ lọc Board đọc/ghi trên URL (giữ nguyên các query param khác như của useNoticeFocus). */
export function useBoardFilters() {
  const [params, setParams] = useSearchParams();
  const filters = useMemo(() => parseBoardFilters(params), [params]);

  const update = useCallback(
    (patch: Partial<BoardFilters>) =>
      setParams((current) => writeBoardFilters(current, { ...parseBoardFilters(current), ...patch }), {
        replace: true,
      }),
    [setParams],
  );

  const clear = useCallback(
    () =>
      setParams(
        (current) => {
          const sort = parseBoardFilters(current).sort;
          return writeBoardFilters(current, { ...parseBoardFilters(new URLSearchParams()), sort });
        },
        { replace: true },
      ),
    [setParams],
  );

  return { filters, update, clear };
}
