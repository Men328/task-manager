import { useCallback } from 'react';

import { selectCurrentProjectId } from '../project/projectSlice';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchReport,
  selectReportActivity,
  selectReportBreakdown,
  selectReportError,
  selectReportLoading,
  selectReportOverview,
  selectReportRange,
  selectReportSeries,
  type ReportRange,
} from './reportSlice';

export function useReport() {
  const dispatch = useAppDispatch();
  const overview = useAppSelector(selectReportOverview);
  const breakdown = useAppSelector(selectReportBreakdown);
  const series = useAppSelector(selectReportSeries);
  const activity = useAppSelector(selectReportActivity);
  const range = useAppSelector(selectReportRange);
  const loading = useAppSelector(selectReportLoading);
  const error = useAppSelector(selectReportError);
  const profileId = useAppSelector((state) => state.session.profileId);
  const projectId = useAppSelector(selectCurrentProjectId);

  const applyRange = useCallback(
    (next: Partial<ReportRange>) => {
      if (!profileId) {
        return;
      }
      void dispatch(
        fetchReport({
          profileId,
          projectId,
          from: next.from !== undefined ? next.from : range.from,
          to: next.to !== undefined ? next.to : range.to,
          interval: next.interval ?? range.interval,
          includeArchived: next.includeArchived ?? range.includeArchived,
        }),
      );
    },
    [dispatch, profileId, projectId, range],
  );

  const refresh = useCallback(() => {
    if (!profileId) {
      return;
    }
    void dispatch(fetchReport({ profileId, projectId, ...range }));
  }, [dispatch, profileId, projectId, range]);

  return {
    overview,
    breakdown,
    series,
    activity,
    range,
    loading,
    error,
    profileId,
    projectId,
    applyRange,
    refresh,
  };
}
