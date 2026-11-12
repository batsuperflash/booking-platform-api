package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestConcurrentBookingSameSlot(t *testing.T) {

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://booking:booking@localhost:5433/booking?sslmode=disable"
	}

	ctx := context.Background()

	db, err := NewPostgres(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	resource, err := db.CreateResource(ctx, CreateResourceParams{
		Name:        fmt.Sprintf("Test Room %d", time.Now().UnixNano()),
		Type:        "room",
		Description: "room for concurrent booking rent",
	})
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	defer func() {
		_, cleanupErr := db.Pool.Exec(
			ctx,
			"DELETE FROM resources WHERE id = $1::uuid",
			resource.ID,
		)
		if cleanupErr != nil {
			t.Errorf("failed to clean up test resource: %v", cleanupErr)
		}
	}()

	slot, err := db.CreateSlot(ctx, CreateSlotParams{
		ResourceID: resource.ID,
		StartsAt:   time.Now().Add(24 * time.Hour),
		EndsAt:     time.Now().Add(25 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create slot: %v", err)
	}

	const attempts = 20

	var wg sync.WaitGroup
	var mu sync.Mutex

	successCount := 0
	conflictCount := 0
	unexpectedErrors := make([]error, 0)

	start := make(chan struct{})

	for i := 0; i < attempts; i++ {

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			<-start

			_, err := db.CreateBooking(ctx, CreateBookingParams{
				SlotID:   slot.ID,
				UserName: fmt.Sprintf("User %d", i),
			})
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successCount++
				return
			}

			if errors.Is(err, ErrSlotAlreadyBooked) {
				conflictCount++
				return
			}
			unexpectedErrors = append(unexpectedErrors, err)
		}(i)
	}

	close(start)
	wg.Wait()

	if len(unexpectedErrors) > 0 {
		t.Fatalf("unexpected errors: %v", unexpectedErrors)
	}

	if successCount != 1 {
		t.Fatalf("expected 1 successful booking, got %d", successCount)
	}

	if conflictCount != attempts-1 {
		t.Fatalf("expected %d conflicts, got %d", attempts-1, conflictCount)
	}

}
