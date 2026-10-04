import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  getActivityReport,
  getReportOverview,
  getStatusBreakdown,
  getTaskTimeSeries,
} from '../../api/report';
import i18n from '../../i18n';
import type {
  ReportActivity,
  ReportInterval,
  ReportOverview,
  ReportStatusBreakdown,
  ReportTimeSeries,
} from '../../types';
import type { RootState } from '../store';

export interface ReportRange {
  from?: string;
  to?: string;
  interval: ReportInterval;
  includeArchived: boolean;
}

interface FetchReportArgs extends ReportRange {
  profileId: string;
}

interface ReportData {
  overview: ReportOverview;
  breakdown: ReportStatusBreakdown;
  series: ReportTimeSeries;
  activity: ReportActivity;
}

export const fetchReport = createAsyncThunk<ReportData, FetchReportArgs, { rejectValue: string }>(
  'report/fetchReport',
  async (params, { rejectWithValue }) => {
    const common = {
      profileId: params.profileId,
      from: params.from,
      to: params.to,
      includeArchived: params.includeArchived,
    };
    try {
      const [overview, breakdown, series, activity] = await Promise.all([
        getReportOverview(common),
        getStatusBreakdown(common),
        getTaskTimeSeries({ ...common, interval: params.interval }),
        getActivityReport(common),
      ]);
      return { overview, breakdown, series, activity };
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export interface ReportState {
  overview: ReportOverview | null;
  breakdown: ReportStatusBreakdown;
  series: ReportTimeSeries | null;
  activity: ReportActivity | null;
  range: ReportRange;
  loading: boolean;
  error: string | null;
}

const initialState: ReportState = {
  overview: null,
  breakdown: { items: [], total: 0 },
  series: null,
  activity: null,
  range: {
    interval: 'REPORT_INTERVAL_DAY',
    includeArchived: false,
  },
  loading: false,
  error: null,
};

const reportSlice = createSlice({
  name: 'report',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchReport.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchReport.fulfilled, (state, action) => {
        state.loading = false;
        state.overview = action.payload.overview;
        state.breakdown = action.payload.breakdown;
        state.series = action.payload.series;
        state.activity = action.payload.activity;
        state.range = {
          from: action.meta.arg.from,
          to: action.meta.arg.to,
          interval: action.meta.arg.interval,
          includeArchived: action.meta.arg.includeArchived,
        };
      })
      .addCase(fetchReport.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadReport');
      });
  },
});

export const reportReducer = reportSlice.reducer;

export const selectReportOverview = (state: RootState) => state.report.overview;
export const selectReportBreakdown = (state: RootState) => state.report.breakdown;
export const selectReportSeries = (state: RootState) => state.report.series;
export const selectReportActivity = (state: RootState) => state.report.activity;
export const selectReportRange = (state: RootState) => state.report.range;
export const selectReportLoading = (state: RootState) => state.report.loading;
export const selectReportError = (state: RootState) => state.report.error;
