package service

import (
	"context"
	"log/slog"
)

func seedDefaultLifecycle(ctx context.Context, seeder DefaultLifecycleSeeder, profileID string) {
	if seeder == nil {
		return
	}
	if err := seeder.SeedDefaultStatuses(ctx, profileID); err != nil {
		slog.Warn("tạo bộ status/lifecycle mặc định thất bại; có thể gọi lại POST /v1/statuses/seed",
			"profile_id", profileID,
			"error", err,
		)
	}
}
