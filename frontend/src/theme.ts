import {
  createTheme,
  localStorageColorSchemeManager,
  type MantineColorsTuple,
} from '@mantine/core';

/**
 * Chế độ màu: `auto` theo hệ điều hành, `light`/`dark` do người dùng chọn.
 * Giá trị được lưu ở localStorage dưới `COLOR_SCHEME_STORAGE_KEY`.
 *
 * Lưu ý: `index.html` có inline script đọc cùng key này để set
 * `data-mantine-color-scheme` trước khi app render (tránh nháy trắng/đen).
 * Đổi key thì phải đổi cả trong `index.html`.
 */
export type AppColorScheme = 'light' | 'dark' | 'auto';

export const COLOR_SCHEME_STORAGE_KEY = 'tm-color-scheme';

export const DEFAULT_COLOR_SCHEME: AppColorScheme = 'auto';

export const colorSchemeManager = localStorageColorSchemeManager({
  key: COLOR_SCHEME_STORAGE_KEY,
});

/**
 * Token ngữ nghĩa dùng trong TSX. Giá trị thật (light/dark) nằm ở
 * `src/styles/tokens.css`; ở đây chỉ trỏ tới CSS variable nên component
 * tự đổi màu khi color scheme thay đổi mà không cần re-mount.
 *
 * Đổi màu = sửa `styles/tokens.css`.
 */
export const tokens = {
  appBg: 'var(--tm-app-bg)',
  surface: 'var(--tm-surface)',
  columnBg: 'var(--tm-column-bg)',
  border: 'var(--tm-border)',
  borderStrong: 'var(--tm-border-strong)',
  text: 'var(--tm-text)',
  textMuted: 'var(--tm-text-muted)',
  textFaint: 'var(--tm-text-faint)',
  navText: 'var(--tm-nav-text)',
  accentTag: 'var(--tm-accent-tag)',
  brand: 'var(--tm-brand)',
  brandLight: 'var(--tm-brand-light)',
  brandFrom: 'var(--tm-brand-gradient-from)',
  brandTo: 'var(--tm-brand-gradient-to)',
  navActiveBg: 'var(--tm-nav-active-bg)',
  progressTrack: 'var(--tm-progress-track)',
  progressFill: 'var(--tm-progress-fill)',
  danger: 'var(--tm-danger)',
} as const;

/** Palette thương hiệu (indigo) — brand[6] là màu chính. */
const brand: MantineColorsTuple = [
  '#eef0ff',
  '#dee2ff',
  '#b9c0ff',
  '#909aff',
  '#6d7aff',
  '#5566ff',
  '#4353e8',
  '#3644c9',
  '#2b37a5',
  '#1e2879',
];

/**
 * Palette `dark` bám theo token dark ở `styles/tokens.css` để component
 * Mantine (input, menu, modal, button variant="default"…) khớp với phần
 * CSS module tự style.
 */
const dark: MantineColorsTuple = [
  '#e7e9f2',
  '#c8cddb',
  '#9aa1b5',
  '#6e7688',
  '#333a4d',
  '#1e2330',
  '#171a24',
  '#0f1117',
  '#0b0d13',
  '#070810',
];

const fontFamily =
  '"Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif';

export const theme = createTheme({
  primaryColor: 'brand',
  colors: { brand, dark },
  primaryShade: 6,
  fontFamily,
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
  headings: { fontFamily, fontWeight: '700' },
  defaultRadius: 'md',
  black: '#16192c',
  components: {
    Button: {
      defaultProps: { radius: 'md' },
    },
    Paper: {
      defaultProps: { radius: 'lg' },
    },
    Modal: {
      defaultProps: { radius: 'lg', centered: true },
    },
  },
});

export default theme;
