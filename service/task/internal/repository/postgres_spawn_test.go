package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/task/internal/model"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TASK_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TASK_TEST_DATABASE_URL chưa đặt: bỏ qua test cần Postgres")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedRecurringTask(t *testing.T, ctx context.Context, pool *pgxpool.Pool, repo *PostgresTaskRepository) model.Task {
	t.Helper()
	var profileID, projectID, statusID string
	err := pool.QueryRow(ctx, `
		SELECT p.profile_id::text, p.id::text, s.id::text
		FROM project.projects p JOIN task.task_statuses s ON s.profile_id = p.profile_id
		WHERE NOT p.is_archived LIMIT 1`).Scan(&profileID, &projectID, &statusID)
	if err != nil {
		t.Skipf("không có profile/project/status để test: %v", err)
	}
	due := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	prev, err := repo.Create(ctx, model.Task{
		ProfileID: profileID, ProjectID: projectID, StatusID: statusID, Title: "[spawn-it] prev",
		DueAt: &due, RecurrenceRule: "FREQ=DAILY", RecurrenceTZ: "Asia/Ho_Chi_Minh",
	})
	if err != nil {
		t.Fatalf("create prev: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM task.tasks WHERE title LIKE '[spawn-it]%'`)
	})
	return prev
}

func nextOf(prev model.Task) model.Task {
	due := prev.DueAt.Add(24 * time.Hour)
	return model.Task{
		ProfileID: prev.ProfileID, ProjectID: prev.ProjectID, StatusID: prev.StatusID, Title: "[spawn-it] next",
		DueAt: &due, RecurrenceRule: prev.RecurrenceRule, RecurrenceTZ: prev.RecurrenceTZ, RecurrenceSeriesID: prev.ID,
	}
}

func countSpawned(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM task.tasks WHERE title = '[spawn-it] next'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestPostgresSpawnNextOccurrenceLinksOnce(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	repo := NewPostgresTaskRepository(pool)
	prev := seedRecurringTask(t, ctx, pool, repo)

	created, err := repo.SpawnNextOccurrence(ctx, prev.ID, "", nextOf(prev))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	got, _ := repo.Get(ctx, prev.ID)
	if got.RecurrenceNextID != created.ID || created.RecurrenceSeriesID != prev.ID {
		t.Fatalf("link = %q, series = %q", got.RecurrenceNextID, created.RecurrenceSeriesID)
	}

	if _, err := repo.SpawnNextOccurrence(ctx, prev.ID, "", nextOf(prev)); !errors.Is(err, model.ErrAlreadyExists) {
		t.Fatalf("second spawn must be rejected, got %v", err)
	}
	if n := countSpawned(t, ctx, pool); n != 1 {
		t.Fatalf("rejected spawn must roll back its insert, found %d copies", n)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	again, err := repo.SpawnNextOccurrence(ctx, prev.ID, created.ID, nextOf(prev))
	if err != nil {
		t.Fatalf("respawn after the linked copy was deleted: %v", err)
	}
	if got, _ := repo.Get(ctx, prev.ID); got.RecurrenceNextID != again.ID {
		t.Fatalf("link must move to the new copy, got %q", got.RecurrenceNextID)
	}
}

func TestPostgresSpawnNextOccurrenceConcurrent(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	repo := NewPostgresTaskRepository(pool)
	prev := seedRecurringTask(t, ctx, pool, repo)

	const workers = 5
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.SpawnNextOccurrence(ctx, prev.ID, "", nextOf(prev))
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok := 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, model.ErrAlreadyExists):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || countSpawned(t, ctx, pool) != 1 {
		t.Fatalf("exactly one concurrent spawn must win, got %d successes and %d rows", ok, countSpawned(t, ctx, pool))
	}
}
