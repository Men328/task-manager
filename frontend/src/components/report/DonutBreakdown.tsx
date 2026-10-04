import { Text } from '@mantine/core';

import { formatPercent } from '../../lib/format';
import classes from './report.module.css';

export interface DonutSlice {
  id: string;
  label: string;
  value: number;
  color: string;
  percentage: number;
}

const SIZE = 190;
const THICKNESS = 26;
const RADIUS = (SIZE - THICKNESS) / 2;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

export function DonutBreakdown({
  slices,
  total,
  centerLabel,
  emptyLabel,
  ariaLabel,
}: {
  slices: DonutSlice[];
  total: number;
  centerLabel: string;
  emptyLabel: string;
  ariaLabel: string;
}) {
  let offset = 0;
  const segments = slices.map((slice) => {
    const length = total > 0 ? (slice.value / total) * CIRCUMFERENCE : 0;
    const segment = (
      <circle
        key={slice.id}
        cx={SIZE / 2}
        cy={SIZE / 2}
        r={RADIUS}
        fill="none"
        stroke={slice.color}
        strokeWidth={THICKNESS}
        strokeDasharray={`${length} ${CIRCUMFERENCE - length}`}
        strokeDashoffset={-offset}
        transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}
      />
    );
    offset += length;
    return segment;
  });

  return (
    <div className={classes.donutWrap}>
      <svg width={SIZE} height={SIZE} role="img" aria-label={ariaLabel}>
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={RADIUS}
          fill="none"
          stroke="var(--tm-progress-track)"
          strokeWidth={THICKNESS}
        />
        {segments}
        <text x={SIZE / 2} y={SIZE / 2 - 2} textAnchor="middle" className={classes.donutCenterValue}>
          {total}
        </text>
        <text x={SIZE / 2} y={SIZE / 2 + 17} textAnchor="middle" className={classes.donutCenterLabel}>
          {centerLabel}
        </text>
      </svg>

      {slices.length === 0 ? (
        <Text className={classes.empty}>{emptyLabel}</Text>
      ) : (
        <div className={classes.breakdownList}>
          {slices.map((slice) => (
            <div key={slice.id} className={classes.breakdownRow}>
              <span className={classes.legendSwatch} style={{ background: slice.color }} />
              <span className={classes.breakdownName} title={slice.label}>
                {slice.label}
              </span>
              <span className={classes.breakdownCount}>{slice.value}</span>
              <span className={classes.breakdownPercent}>{formatPercent(slice.percentage)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
