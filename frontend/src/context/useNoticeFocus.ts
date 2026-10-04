import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';

/**
 * Đọc `?focus=<id>` do notice điều hướng tới, tìm đối tượng tương ứng rồi mở chi tiết.
 *
 * Id chỉ được xoá khỏi URL **sau khi** đã xử lý xong. Trước đây hook xoá URL ngay
 * lập tức, khiến effect cleanup chạy trước khi promise resolve và `open` bị bỏ qua
 * (noti chỉ chuyển tab mà không mở chi tiết). Việc xoá muộn + cờ `handled` giữ cho
 * thao tác mở luôn xảy ra, kể cả khi phải fetch đối tượng từ API.
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

  // Giữ setParams trong ref để effect không phụ thuộc identity của nó (đổi mỗi render).
  const setParamsRef = useRef(setParams);
  setParamsRef.current = setParams;

  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const handledRef = useRef<string | null>(null);

  useEffect(() => {
    if (!focusId) {
      // Cho phép cùng một notice được click lại ở lần sau.
      handledRef.current = null;
      return;
    }
    if (handledRef.current === focusId) {
      return;
    }
    handledRef.current = focusId;

    const clearFocus = () => {
      const next = new URLSearchParams(window.location.search);
      if (!next.has('focus')) {
        return;
      }
      next.delete('focus');
      setParamsRef.current(next, { replace: true });
    };

    Promise.resolve(resolveRef.current(focusId))
      .then((item) => {
        // Bỏ qua nếu component đã unmount hoặc notice khác đã thay thế.
        if (!mountedRef.current || handledRef.current !== focusId) {
          return;
        }
        if (item) {
          openRef.current(item);
        }
        clearFocus();
      })
      .catch(() => {
        if (mountedRef.current && handledRef.current === focusId) {
          clearFocus();
        }
      });
  }, [focusId]);
}
