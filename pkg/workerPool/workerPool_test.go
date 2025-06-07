package workerPool

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("should create new WorkersPool with correct initial values", func(t *testing.T) {
		wp := New(10)
		if wp == nil {
			t.Fatal("Expected WorkersPool instance, got nil")
		}
		if len(wp.workers) != 0 {
			t.Errorf("Expected 0 workers, got %d", len(wp.workers))
		}
		if cap(wp.jobsCh) != 10 {
			t.Errorf("Expected jobs channel capacity 10, got %d", cap(wp.jobsCh))
		}
		if wp.maxJobCount != 10 {
			t.Errorf("Expected maxJobCount 10, got %d", wp.maxJobCount)
		}
	})
}

func TestAddJob(t *testing.T) {
	t.Run("should add job when channel is not full", func(t *testing.T) {
		wp := New(2)
		err := wp.AddJob("test1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		err = wp.AddJob("test2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(wp.jobsCh) != 2 {
			t.Errorf("Expected 2 jobs in channel, got %d", len(wp.jobsCh))
		}
	})

	t.Run("should return error when channel is full", func(t *testing.T) {
		wp := New(1)
		_ = wp.AddJob("test1")
		err := wp.AddJob("test2")
		if err == nil {
			t.Error("Expected error when channel is full, got nil")
		}
		if !strings.Contains(err.Error(), "Channel was full") {
			t.Errorf("Expected 'Channel was full' error, got %v", err)
		}
	})
}

func TestAddWorkers(t *testing.T) {
	t.Run("should add specified number of workers", func(t *testing.T) {
		wp := New(10)
		ctx := context.Background()
		err := wp.AddWorkers(ctx, 3)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(wp.workers) != 3 {
			t.Errorf("Expected 3 workers, got %d", len(wp.workers))
		}
	})

	t.Run("workers should process jobs", func(t *testing.T) {
		wp := New(10)
		ctx := context.Background()
		_ = wp.AddWorkers(ctx, 1)
		_ = wp.AddJob("test job")
		time.Sleep(100 * time.Millisecond)
		wp.Stop()
	})
}

func TestDeleteWorkers(t *testing.T) {
	t.Run("should remove specified number of workers", func(t *testing.T) {
		wp := New(10)
		ctx := context.Background()
		_ = wp.AddWorkers(ctx, 5)
		err := wp.DeleteWorkers(2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(wp.workers) != 3 {
			t.Errorf("Expected 3 workers after deletion, got %d", len(wp.workers))
		}
	})

	t.Run("should return error when count is invalid", func(t *testing.T) {
		wp := New(10)
		ctx := context.Background()
		_ = wp.AddWorkers(ctx, 2)

		tests := []struct {
			name  string
			count int
		}{
			{"negative count", -1},
			{"zero count", 0},
			{"count greater than workers", 3},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := wp.DeleteWorkers(tt.count)
				if err == nil {
					t.Error("Expected error, got nil")
				}
			})
		}
	})
}

func TestStop(t *testing.T) {
	t.Run("should stop all workers and clean up", func(t *testing.T) {
		wp := New(10)
		ctx := context.Background()
		_ = wp.AddWorkers(ctx, 3)
		_ = wp.AddJob("test1")
		_ = wp.AddJob("test2")

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.Stop()
		}()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Error("Stop() did not complete within expected time")
		}
	})
}

func TestWorkerContextCancellation(t *testing.T) {
	t.Run("workers should stop when context is cancelled", func(t *testing.T) {
		wp := New(10)
		ctx, cancel := context.WithCancel(context.Background())
		_ = wp.AddWorkers(ctx, 2)

		cancel()

		time.Sleep(100 * time.Millisecond)

		done := make(chan struct{})
		go func() {
			wp.Stop()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Error("Workers did not stop after context cancellation")
		}
	})
}

func TestLogFileCreation(t *testing.T) {
	t.Run("should create legacy.log file on initialization", func(t *testing.T) {
		_ = os.Remove("legacy.log")

		_ = New(10)

		if _, err := os.Stat("legacy.log"); os.IsNotExist(err) {
			t.Error("Expected legacy.log file to be created, but it doesn't exist")
		}

		_ = os.Remove("legacy.log")
	})
}
