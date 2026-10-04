import { Box, Text } from '@mantine/core';
import { useElementSize } from '@mantine/hooks';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { ReportInterval, ReportTimeSeries } from '../../types';
import classes from './report.module.css';

const HEIGHT = 280;
const PAD = { top: 16, right: 14, bottom: 30, left: 40 };

const CREATED_COLOR = 'var(--tm-brand)';
const COMPLETED_COLOR = '#37b24d';

function niceMax(value: number): number {
  const base = Math.max(1, value);
  return Math.max(4, Math.ceil(base / 4) * 4);
}

function formatBucket(bucket: string, interval: ReportInterval, locale: string): string {
  const date = new Date(`${bucket}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) {
    return bucket;
  }
  if (interval === 'REPORT_INTERVAL_MONTH') {
    return date.toLocaleDateString(locale, { month: 'short', year: '2-digit', timeZone: 'UTC' });
  }
  return date.toLocaleDateString(locale, {
    day: '2-digit',
    month: '2-digit',
    timeZone: 'UTC',
  });
}

function formatDay(bucket: string, locale: string): string {
  const date = new Date(`${bucket}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) {
    return bucket;
  }
  return date.toLocaleDateString(locale, { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' });
}

export function TaskTrendChart({ series }: { series: ReportTimeSeries }) {
  const { t, i18n } = useTranslation();
  const { ref, width } = useElementSize();
  const [active, setActive] = useState<number | null>(null);

  const locale = i18n.resolvedLanguage?.startsWith('vi') ? 'vi-VN' : 'en-GB';
  const points = series.points;
  const count = points.length;

  const maxValue = points.reduce(
    (acc, point) => Math.max(acc, point.created, point.completed),
    0,
  );
  const yMax = niceMax(maxValue);

  const innerW = Math.max(10, (width || 520) - PAD.left - PAD.right);
  const innerH = HEIGHT - PAD.top - PAD.bottom;

  const xAt = (index: number) =>
    count <= 1 ? PAD.left + innerW / 2 : PAD.left + (index * innerW) / (count - 1);
  const yAt = (value: number) => PAD.top + innerH - (value / yMax) * innerH;

  const lineFor = (key: 'created' | 'completed') =>
    points.map((point, index) => `${xAt(index)},${yAt(point[key])}`).join(' ');

  const areaFor = (key: 'created' | 'completed') => {
    if (count === 0) {
      return '';
    }
    const baseline = PAD.top + innerH;
    return `M ${xAt(0)},${baseline} L ${lineFor(key).split(' ').join(' L ')} L ${xAt(count - 1)},${baseline} Z`;
  };

  const ticks = [0, 1, 2, 3, 4].map((step) => (yMax / 4) * step);
  const labelStep = Math.max(1, Math.ceil(count / 7));
  const activePoint = active !== null ? points[active] : undefined;

  const legend = (
    <div className={classes.legend}>
      <span className={classes.legendItem}>
        <span className={classes.legendSwatch} style={{ background: CREATED_COLOR }} />
        {t('reportPage.seriesCreated', { n: series.totalCreated })}
      </span>
      <span className={classes.legendItem}>
        <span className={classes.legendSwatch} style={{ background: COMPLETED_COLOR }} />
        {t('reportPage.seriesCompleted', { n: series.totalCompleted })}
      </span>
    </div>
  );

  return (
    <Box>
      {legend}
      <div ref={ref} className={classes.chartWrap} style={{ marginTop: 10 }}>
        {count === 0 ? (
          <Text className={classes.empty}>{t('reportPage.noTimeSeries')}</Text>
        ) : (
          <>
            <svg width="100%" height={HEIGHT} role="img" aria-label={t('reportPage.trendChartTitle')}>
              {ticks.map((tick) => (
                <g key={tick}>
                  <line
                    x1={PAD.left}
                    x2={PAD.left + innerW}
                    y1={yAt(tick)}
                    y2={yAt(tick)}
                    className={classes.gridLine}
                  />
                  <text
                    x={PAD.left - 8}
                    y={yAt(tick) + 3.5}
                    textAnchor="end"
                    className={classes.axisLabel}
                  >
                    {tick}
                  </text>
                </g>
              ))}

              <path d={areaFor('created')} fill={CREATED_COLOR} opacity={0.08} />
              <path d={areaFor('completed')} fill={COMPLETED_COLOR} opacity={0.08} />

              <polyline
                points={lineFor('created')}
                fill="none"
                stroke={CREATED_COLOR}
                strokeWidth={2.4}
                strokeLinejoin="round"
                strokeLinecap="round"
              />
              <polyline
                points={lineFor('completed')}
                fill="none"
                stroke={COMPLETED_COLOR}
                strokeWidth={2.4}
                strokeLinejoin="round"
                strokeLinecap="round"
              />

              {points.map((point, index) => (
                <g key={point.bucket}>
                  {index % labelStep === 0 || index === count - 1 ? (
                    <text
                      x={xAt(index)}
                      y={HEIGHT - 9}
                      textAnchor="middle"
                      className={classes.axisLabel}
                    >
                      {formatBucket(point.bucket, series.interval, locale)}
                    </text>
                  ) : null}
                </g>
              ))}

              {active !== null && activePoint ? (
                <>
                  <line
                    x1={xAt(active)}
                    x2={xAt(active)}
                    y1={PAD.top}
                    y2={PAD.top + innerH}
                    className={classes.hoverLine}
                  />
                  <circle cx={xAt(active)} cy={yAt(activePoint.created)} r={4} fill={CREATED_COLOR} />
                  <circle
                    cx={xAt(active)}
                    cy={yAt(activePoint.completed)}
                    r={4}
                    fill={COMPLETED_COLOR}
                  />
                </>
              ) : null}

              <rect
                x={PAD.left}
                y={PAD.top}
                width={innerW}
                height={innerH}
                fill="transparent"
                onMouseMove={(event) => {
                  const rect = event.currentTarget.getBoundingClientRect();
                  const ratio = rect.width > 0 ? (event.clientX - rect.left) / rect.width : 0;
                  const index = count <= 1 ? 0 : Math.round(ratio * (count - 1));
                  setActive(Math.max(0, Math.min(count - 1, index)));
                }}
                onMouseLeave={() => setActive(null)}
              />
            </svg>

            {active !== null && activePoint ? (
              <div
                className={classes.tooltip}
                style={{
                  left: Math.min(Math.max(xAt(active), 76), (width || 520) - 76),
                  top: 0,
                }}
              >
                <div className={classes.tooltipTitle}>
                  {formatDay(activePoint.bucket, locale)}
                </div>
                <div className={classes.tooltipRow}>
                  <span>{t('reportPage.legendCreated')}</span>
                  <span className={classes.tooltipValue}>{activePoint.created}</span>
                </div>
                <div className={classes.tooltipRow}>
                  <span>{t('reportPage.legendCompleted')}</span>
                  <span className={classes.tooltipValue}>{activePoint.completed}</span>
                </div>
              </div>
            ) : null}
          </>
        )}
      </div>
    </Box>
  );
}
