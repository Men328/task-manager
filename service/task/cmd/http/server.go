package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/config"
)

func setupLogger(cfg config.Config) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()})))
}

func newGRPCConn(lc fx.Lifecycle, cfg config.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		cfg.GRPCDialTarget(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial gRPC %s: %w", cfg.GRPCDialTarget(), err)
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return conn, nil
}

func newServeMux(cfg config.Config, conn *grpc.ClientConn) (*runtime.ServeMux, error) {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
	)

	ctx := context.Background()
	handlers := []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{
		taskv1.RegisterTaskServiceHandler,
		taskv1.RegisterTaskStatusServiceHandler,
		taskv1.RegisterStatusTransitionServiceHandler,
	}
	for _, register := range handlers {
		if err := register(ctx, mux, conn); err != nil {
			return nil, fmt.Errorf("register gateway handler: %w", err)
		}
	}

	healthClient := healthpb.NewHealthClient(conn)
	if err := mux.HandlePath(http.MethodGet, "/healthz", healthzHandler(cfg.ServiceName, healthClient)); err != nil {
		return nil, fmt.Errorf("register /healthz: %w", err)
	}
	return mux, nil
}

func newHTTPListener(cfg config.Config) (net.Listener, error) {
	return net.Listen("tcp", cfg.HTTPAddr)
}

func newHTTPServer(mux *runtime.ServeMux) *http.Server {
	return &http.Server{
		Handler:           withRequestLog(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func serveHTTP(lc fx.Lifecycle, cfg config.Config, server *http.Server, listener net.Listener) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			slog.Info("HTTP gateway listening",
				"service", cfg.ServiceName,
				"addr", listener.Addr().String(),
				"grpc_target", cfg.GRPCDialTarget(),
			)
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("serve HTTP", "service", cfg.ServiceName, "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

func healthzHandler(serviceName string, client healthpb.HealthClient) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
		w.Header().Set("Content-Type", "application/json")
		if err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"degraded","service":%q}`, serviceName)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","service":%q}`, serviceName)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
