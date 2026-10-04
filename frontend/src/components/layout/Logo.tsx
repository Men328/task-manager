import { useId } from 'react';

interface LogoProps {
  /** Cạnh của logo theo px (mặc định 32). */
  size?: number;
  /** Bo góc trong hệ toạ độ viewBox 64×64 (mặc định 17). */
  radius?: number;
  /** Nhãn truy cập; đổi thành rỗng nếu logo chỉ mang tính trang trí. */
  title?: string;
  className?: string;
}

/**
 * Logo thương hiệu của Task Manager: thẻ công việc xếp lớp + dấu tick gradient.
 * Dùng chung cho sidebar, trang đăng nhập… Gradien có id duy nhất theo `useId`
 * để nhiều instance trên cùng trang không ghi đè lẫn nhau.
 */
export function Logo({ size = 32, radius = 17, title = 'Task Manager', className }: LogoProps) {
  const rawId = useId();
  const gradientId = `tm-logo-${rawId.replace(/[^a-zA-Z0-9_-]/g, '')}`;

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className={className}
      role="img"
      aria-label={title}
      focusable="false"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#4353e8" />
          <stop offset="1" stopColor="#7c3aed" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx={radius} fill={`url(#${gradientId})`} />
      <rect x="27" y="9" width="27" height="24" rx="8" fill="#ffffff" opacity="0.32" />
      <rect x="10" y="19" width="36" height="32" rx="10" fill="#ffffff" />
      <path
        d="M19 35 L26.5 42.5 L39 27"
        fill="none"
        stroke={`url(#${gradientId})`}
        strokeWidth="5.4"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export default Logo;
