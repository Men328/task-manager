/** report service HTTP calls. */

import { buildQuery, request, reportApi } from './client';
import type {
  ReportActivity,
  ReportOverview,
  ReportStatusBreakdown,
  ReportTimeSeries,
  ReportTimeSeriesParams,
  ReportQueryParams,
} from '../types';

const REPORTS_PATH = '/v1/reports';

function toQuery(params: ReportQueryParams): string {
  return buildQuery({
    profile_id: params.profileId,
    from: params.from,
    to: params.to,
    include_archived: params.includeArchived,
  });
}

/** GET /v1/reports/overview */
export function getReportOverview(params: ReportQueryParams): Promise<ReportOverview> {
  return request<unknown>(reportApi, `${REPORTS_PATH}/overview${toQuery(params)}`, {
    method: 'GET',
  }).then((payload) => unwrap<ReportOverview>(payload, 'overview'));
}

/** GET /v1/reports/status-breakdown */
export function getStatusBreakdown(params: ReportQueryParams): Promise<ReportStatusBreakdown> {
  return request<unknown>(reportApi, `${REPORTS_PATH}/status-breakdown${toQuery(params)}`, {
    method: 'GET',
  }).then((payload) => normalizeBreakdown(payload));
}

/** GET /v1/reports/timeseries */
export function getTaskTimeSeries(params: ReportTimeSeriesParams): Promise<ReportTimeSeries> {
  const query = buildQuery({
    profile_id: params.profileId,
    from: params.from,
    to: params.to,
    include_archived: params.includeArchived,
    interval: params.interval,
  });
  return request<ReportTimeSeries>(reportApi, `${REPORTS_PATH}/timeseries${query}`, {
    method: 'GET',
  });
}

/** GET /v1/reports/activity — thống kê sự kiện + lịch + backlog. */
export function getActivityReport(params: ReportQueryParams): Promise<ReportActivity> {
  return request<ReportActivity>(reportApi, `${REPORTS_PATH}/activity${toQuery(params)}`, {
    method: 'GET',
  });
}

function normalizeBreakdown(payload: unknown): ReportStatusBreakdown {
  if (payload !== null && typeof payload === 'object') {
    const record = payload as Record<string, unknown>;
    const items = Array.isArray(record.items) ? record.items : [];
    return {
      items: items as ReportStatusBreakdown['items'],
      total: typeof record.total === 'number' ? record.total : 0,
    };
  }
  return { items: [], total: 0 };
}

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
