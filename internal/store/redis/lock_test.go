package redis_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	redisstore "github.com/DanielPopoola/fairqueue/internal/store/redis"
	"github.com/DanielPopoola/fairqueue/internal/store/testhelpers"
	"github.com/redis/go-redis/v9"
)

var (
	testClient *redis.Client
	testCtx    = context.Background()
)

func TestMain(m *testing.M) {
	rc, err := testhelpers.NewRedisInstance(testCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setting up redis: %v\n", err)
		os.Exit(1)
	}
	defer rc.Close(testCtx)

	testClient = rc.Client
	os.Exit(m.Run())
}

func TestLockStore_Acquire_SecondCallFails(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	store := redisstore.NewLockStore(client, 30*time.Second)
	claimID := "claim-123"

	// First acquire succeeds
	if err := store.Acquire(testCtx, claimID); err != nil {
		t.Fatalf("expected first acquire to succeed, got: %v", err)
	}

	// Second acquire on same ID fails
	if err := store.Acquire(testCtx, claimID); err != redisstore.ErrLockNotAcquired {
		t.Fatalf("expected ErrLockNotAcquired, got: %v", err)
	}
}

func TestLockStore_Acquire_SucceedsAfterRelease(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	store := redisstore.NewLockStore(client, 30*time.Second)
	claimID := "claim-456"

	if err := store.Acquire(testCtx, claimID); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if err := store.Release(testCtx, claimID); err != nil {
		t.Fatalf("release: %v", err)
	}

	// Should be acquirable again after release
	if err := store.Acquire(testCtx, claimID); err != nil {
		t.Fatalf("expected acquire after release to succeed, got: %v", err)
	}
}

func TestLockStore_Acquire_SucceedsAfterTTLExpiry(t *testing.T) {
	testhelpers.FlushRedis(testCtx, t, testClient)

	client, err := redisstore.NewClient(testCtx, testClient)
	if err != nil {
		t.Fatalf("creating redis client: %v", err)
	}

	// Very short TTL for testing
	store := redisstore.NewLockStore(client, 100*time.Millisecond)
	claimID := "claim-789"

	if err := store.Acquire(testCtx, claimID); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(200 * time.Millisecond)

	// Should be acquirable again after TTL
	if err := store.Acquire(testCtx, claimID); err != nil {
		t.Fatalf("expected acquire after TTL expiry to succeed, got: %v", err)
	}
}
