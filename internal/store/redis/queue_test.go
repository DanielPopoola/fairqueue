package redis_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
	"github.com/redis/go-redis/v9"
)

func TestQueueStore_Join_DuplicatePrevented(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewQueueStore(client)

	eventID := "event-1"
	userID := "user-1"

	if err := store.Join(testCtx, eventID, userID); err != nil {
		t.Fatalf("first join: %v", err)
	}

	if err := store.Join(testCtx, eventID, userID); err != domain.ErrAlreadyInQueue {
		t.Fatalf("expected ErrAlreadyInQueue, got: %v", err)
	}
}

func TestQueueStore_Join_AdmittedUserCannotRejoin(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewQueueStore(client)

	eventID := "event-2"
	userID := "user-2"

	// Join and admit
	store.Join(testCtx, eventID, userID)
	store.AdmitNextBatch(testCtx, eventID, 1)

	// User is now in admitted ZSET — joining again should fail
	if err := store.Join(testCtx, eventID, userID); err != domain.ErrAlreadyInQueue {
		t.Fatalf("expected ErrAlreadyInQueue for admitted user, got: %v", err)
	}
}

func TestQueueStore_AdmitNextBatch_AtomicUnderConcurrency(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewQueueStore(client)

	eventID := "event-3"

	// Add 20 users to the waiting queue
	for i := range 20 {
		userID := fmt.Sprintf("user-%d", i)
		if err := store.Join(testCtx, eventID, userID); err != nil {
			t.Fatalf("joining user %d: %v", i, err)
		}
	}

	// Two goroutines both try to admit 10 users simultaneously
	var wg sync.WaitGroup
	results := make([][]string, 2)

	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			admitted, err := store.AdmitNextBatch(testCtx, eventID, 10)
			if err != nil {
				t.Errorf("admitting batch %d: %v", idx, err)
				return
			}
			results[idx] = admitted
		}(i)
	}

	wg.Wait()

	// Combine results and check no user appears twice
	seen := make(map[string]bool)
	for _, batch := range results {
		for _, userID := range batch {
			if seen[userID] {
				t.Fatalf("user %s admitted twice — Lua script is not atomic", userID)
			}
			seen[userID] = true
		}
	}

	// All 20 users should have been admitted exactly once
	if len(seen) != 20 {
		t.Fatalf("expected 20 unique admitted users, got %d", len(seen))
	}
}

func TestQueueStore_EvictExpiredAdmitted(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewQueueStore(client)

	eventID := "event-4"

	// Add and admit two users
	store.Join(testCtx, eventID, "user-fresh")
	store.Join(testCtx, eventID, "user-expired")
	store.AdmitNextBatch(testCtx, eventID, 2)

	// Backdate user-expired's admission score in Redis
	// by directly setting a score older than the TTL
	expiredScore := float64(time.Now().Add(-10 * time.Minute).UnixNano())
	testClient.ZAdd(testCtx, fmt.Sprintf("admitted:%s", eventID), redis.Z{
		Score:  expiredScore,
		Member: "user-expired",
	})

	// Evict with 5 minute TTL — only user-expired should be removed
	evicted, err := store.EvictExpiredAdmitted(testCtx, eventID, 5*time.Minute)
	if err != nil {
		t.Fatalf("evicting: %v", err)
	}

	if len(evicted) != 1 {
		t.Fatalf("expected 1 evicted user, got %d", len(evicted))
	}
	if evicted[0] != "user-expired" {
		t.Fatalf("expected user-expired to be evicted, got %s", evicted[0])
	}

	// user-fresh should still be admitted
	admitted, err := store.IsAdmitted(testCtx, eventID, "user-fresh")
	if err != nil {
		t.Fatalf("checking admission: %v", err)
	}
	if !admitted {
		t.Fatal("expected user-fresh to still be admitted")
	}
}

func TestInventoryStore_SetNX_DoesNotOverwrite(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewInventoryStore(client)

	eventID := "event-5"

	// First Set succeeds
	ok, err := store.Set(testCtx, eventID, 100)
	if err != nil {
		t.Fatalf("first set: %v", err)
	}
	if !ok {
		t.Fatal("expected first set to return true")
	}

	// Second Set on same key returns false — does not overwrite
	ok, err = store.Set(testCtx, eventID, 999)
	if err != nil {
		t.Fatalf("second set: %v", err)
	}
	if ok {
		t.Fatal("expected second set to return false")
	}

	// Value should still be 100
	val, err := store.Get(testCtx, eventID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != 100 {
		t.Fatalf("expected 100, got %d", val)
	}
}

func TestInventoryStore_ForceSet_Overwrites(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, _ := redisstore.NewClient(testCtx, testClient)
	store := redisstore.NewInventoryStore(client)

	eventID := "event-6"

	store.Set(testCtx, eventID, 100)

	// ForceSet should overwrite regardless
	if err := store.ForceSet(testCtx, eventID, 50); err != nil {
		t.Fatalf("force set: %v", err)
	}

	val, err := store.Get(testCtx, eventID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != 50 {
		t.Fatalf("expected 50, got %d", val)
	}
}
