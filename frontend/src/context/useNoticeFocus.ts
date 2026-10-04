import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';

/**
 * Đọc `?focus=<id>` do notice điều hướng tới, tìm đối tượng tương ứng rồi mở chi tiết.
 * Id được xoá khỏi URL ngay sau khi xử lý để refresh không mở lại modal.
 */
export function useNoticeFocus<T>(
  resolve: (id: string) => Promise<T | null> | T | null,
  open: (item: T) => void,
): void {
  const [params, setParams] = useSearchParams();
  const focusId = params.get('focus');
  const resolveRef = useRef(resolve);
  const openRef = useRef(open);
  resolveRef.current = resolve;
  openRef.current = open;

  useEffect(() => {
    if (!focusId) {
      return;
    }

    let cancelled = false;
    Promise.resolve(resolveRef.current(focusId))
      .then((item) => {
        if (!cancelled && item) {
          openRef.current(item);
        }
      })
      .catch(() => undefined);

    const next = new URLSearchParams(window.location.search);
    next.delete('focus');
    setParams(next, { replace: true });

    return () => {
      cancelled = true;
    };
  }, [focusId, setParams]);
}
