import { useTranslation } from 'react-i18next';

import { statusColor } from '../../lib/tokens';
import type { ReportStatusBreakdown } from '../../types';
import { DonutBreakdown, type DonutSlice } from './DonutBreakdown';

export function StatusBreakdownChart({ breakdown }: { breakdown: ReportStatusBreakdown }) {
  const { t } = useTranslation();

  const slices: DonutSlice[] = breakdown.items.map((item, index) => ({
    id: item.statusId,
    label: item.statusName || item.statusSlug,
    value: item.count,
    color: statusColor(item.color, index),
    percentage: item.percentage,
  }));

  return (
    <DonutBreakdown
      slices={slices}
      total={breakdown.total}
      centerLabel={t('reportPage.donutTotal')}
      emptyLabel={t('reportPage.noStatusData')}
      ariaLabel={t('reportPage.statusChartTitle')}
    />
  );
}
