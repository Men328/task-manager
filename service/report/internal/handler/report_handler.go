package handler

import (
	"context"

	reportv1 "taskmanager/common/gen/go/report/v1"
	"taskmanager/service/report/internal/dependency"
	"taskmanager/service/report/internal/service"
)

type ReportHandler struct {
	reportv1.UnimplementedReportServiceServer
	reports service.ReportService
}

func NewReportHandler(reports service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) GetReportOverview(ctx context.Context, req *reportv1.GetReportOverviewRequest) (*reportv1.GetReportOverviewResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	query := dependency.QueryFromParts(req.GetProfileId(), req.GetFrom(), req.GetTo(), req.GetIncludeArchived())
	overview, err := h.reports.Overview(ctx, query)
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &reportv1.GetReportOverviewResponse{Overview: dependency.OverviewToProto(overview)}, nil
}

func (h *ReportHandler) GetStatusBreakdown(ctx context.Context, req *reportv1.GetStatusBreakdownRequest) (*reportv1.GetStatusBreakdownResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	query := dependency.QueryFromParts(req.GetProfileId(), req.GetFrom(), req.GetTo(), req.GetIncludeArchived())
	breakdown, err := h.reports.StatusBreakdown(ctx, query)
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return dependency.StatusBreakdownToProto(breakdown), nil
}

func (h *ReportHandler) GetTaskTimeSeries(ctx context.Context, req *reportv1.GetTaskTimeSeriesRequest) (*reportv1.GetTaskTimeSeriesResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	query := dependency.QueryFromParts(req.GetProfileId(), req.GetFrom(), req.GetTo(), req.GetIncludeArchived())
	series, err := h.reports.TimeSeries(ctx, query, dependency.IntervalFromProto(req.GetInterval()))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return dependency.TimeSeriesToProto(series), nil
}

func (h *ReportHandler) GetActivityReport(ctx context.Context, req *reportv1.GetActivityReportRequest) (*reportv1.GetActivityReportResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	query := dependency.QueryFromParts(req.GetProfileId(), req.GetFrom(), req.GetTo(), false)
	report, err := h.reports.Activity(ctx, query)
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return dependency.ActivityToProto(report), nil
}
