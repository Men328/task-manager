package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/config"
	"taskmanager/service/task/internal/handler"
)

func setupLogger(cfg config.Config) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()})))
}

func newListener(cfg config.Config) (net.Listener, error) {
	return net.Listen("tcp", cfg.GRPCAddr)
}

func newGRPCServer(
	taskHandler *handler.TaskHandler,
	statusHandler *handler.TaskStatusHandler,
	transitionHandler *handler.StatusTransitionHandler,
) *grpc.Server {
	server := grpc.NewServer()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	reflection.Register(server)
	taskv1.RegisterTaskServiceServer(server, taskHandler)
	taskv1.RegisterTaskStatusServiceServer(server, statusHandler)
	taskv1.RegisterStatusTransitionServiceServer(server, transitionHandler)
	return server
}

func serveGRPC(lc fx.Lifecycle, cfg config.Config, server *grpc.Server, listener net.Listener) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			slog.Info("gRPC listening", "service", cfg.ServiceName, "addr", listener.Addr().String())
			go func() {
				if err := server.Serve(listener); err != nil {
					slog.Error("serve gRPC", "service", cfg.ServiceName, "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopped := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-ctx.Done():
				server.Stop()
			}
			return nil
		},
	})
}
