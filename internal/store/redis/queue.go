package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	waitingKeyFmt  = "waiting:%s"  // waiting:{event_id}
	admittedKeyFmt = "admitted:%s" // admitted:{event_id}
)

// joinScript atomically checks both ZSETs and adds the user
// to the waiting queue only if they are not already present.
//
// Returns:
//
//	0 → already in waiting ZSET
//	1 → already in admitted ZSET
//	2 → successfully added
var joinScript = redis.NewScript(`
	local waiting  = KEYS[1]
	local admitted = KEYS[2]
	local userID   = ARGV[1]
	local score    = tonumber(ARGV[2])

	if redis.call("ZSCORE", waiting, userID) ~= false then
		return 0
	end

	if redis.call("ZSCORE", admitted, userID) ~= false then
		return 1
	end

	redis.call("ZADD", waiting, score, userID)
	return 2
`)

// admitBatchScript atomically moves the next n users from
// waiting to admitted in a single Redis operation.
//
// Returns a flat list of admitted userIDs,
// or an empty list if the waiting queue is empty.
var admitBatchScript = redis.NewScript(`
	local waiting   = KEYS[1]
	local admitted  = KEYS[2]
	local batchSize = tonumber(ARGV[1])
	local admScore  = tonumber(ARGV[2])

	local results = redis.call("ZPOPMIN", waiting, batchSize)

	if #results == 0 then
		return {}
	end

	local userIDs = {}
	local zaddArgs = {}

	-- ZPOPMIN returns alternating member/score pairs:
	-- results[1]=userID, results[2]=score, results[3]=userID ...
	-- We only need the userIDs (odd indices).
	for i = 1, #results, 2 do
		local userID = results[i]
		table.insert(userIDs, userID)
		table.insert(zaddArgs, admScore)
		table.insert(zaddArgs, userID)
	end

	-- Unpack zaddArgs as variadic args to ZADD.
	-- redis.call requires individual args, not a table,
	-- so we use unpack() to expand the table.
	redis.call("ZADD", admitted, unpack(zaddArgs))

	return userIDs
`)

// QueueStore manages the virtual waiting queue using two Redis ZSETs.
type QueueStore struct {
	client *Client
}

func NewQueueStore(client *Client) *QueueStore {
	return &QueueStore{client: client}
}

func (s *QueueStore) waitingKey(eventID string) string {
	return fmt.Sprintf(waitingKeyFmt, eventID)
}

func (s *QueueStore) admittedKey(eventID string) string {
	return fmt.Sprintf(admittedKeyFmt, eventID)
}

// Join atomically checks both ZSETs and adds the user to the
// waiting queue if they are not already present in either.
func (s *QueueStore) Join(ctx context.Context, eventID, userID string) error {
	keys := []string{
		s.waitingKey(eventID),
		s.admittedKey(eventID),
	}
	args := []any{
		userID,
		time.Now().UnixNano(),
	}

	res, err := joinScript.Run(ctx, s.client.rdb, keys, args...).Int()
	if err != nil {
		return fmt.Errorf("joining queue: %w", err)
	}

	switch res {
	case 0, 1:
		return domain.ErrAlreadyInQueue
	case 2:
		return nil
	default:
		return fmt.Errorf("unexpected join result: %d", res)
	}
}

// AdmitNextBatch atomically moves the next n longest-waiting users
// from the waiting ZSET to the admitted ZSET.
// Returns the userIDs of admitted users so the service layer can
// generate tokens and notify them via WebSocket.
func (s *QueueStore) AdmitNextBatch(ctx context.Context, eventID string, n int64) ([]string, error) {
	keys := []string{
		s.waitingKey(eventID),
		s.admittedKey(eventID),
	}
	args := []any{
		n,
		time.Now().UnixNano(),
	}

	res, err := admitBatchScript.Run(ctx, s.client.rdb, keys, args...).StringSlice()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("admitting batch: %w", err)
	}

	return res, nil
}

// GetPosition returns the zero-based position of a user in the waiting queue.
// Returns -1 if the user is not in the waiting queue.
func (s *QueueStore) GetPosition(ctx context.Context, eventID, userID string) (int64, error) {
	pos, err := s.client.rdb.ZRank(ctx, s.waitingKey(eventID), userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return -1, nil
		}
		return 0, fmt.Errorf("getting queue position: %w", err)
	}
	return pos, nil
}

// IsAdmitted returns true if the user is currently in the admitted ZSET.
func (s *QueueStore) IsAdmitted(ctx context.Context, eventID, userID string) (bool, error) {
	_, err := s.client.rdb.ZScore(ctx, s.admittedKey(eventID), userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("checking admission status: %w", err)
	}
	return true, nil
}

// RemoveAdmitted removes a user from the admitted ZSET.
// Called when a user successfully claims or their admission window expires.
func (s *QueueStore) RemoveAdmitted(ctx context.Context, eventID, userID string) error {
	if err := s.client.rdb.ZRem(ctx, s.admittedKey(eventID), userID).Err(); err != nil {
		return fmt.Errorf("removing from admitted: %w", err)
	}
	return nil
}

// RemoveWaiting removes a user from the waiting ZSET.
// Called when a user explicitly abandons the queue.
func (s *QueueStore) RemoveWaiting(ctx context.Context, eventID, userID string) error {
	if err := s.client.rdb.ZRem(ctx, s.waitingKey(eventID), userID).Err(); err != nil {
		return fmt.Errorf("removing from waiting: %w", err)
	}
	return nil
}

// GetExpiredAdmitted returns userIDs whose admission window has passed.
// The eviction worker uses this to clean up users who were admitted
// but never completed a claim.
func (s *QueueStore) GetExpiredAdmitted(ctx context.Context, eventID string, admissionTTL time.Duration) ([]string, error) {
	cutoff := fmt.Sprintf("%d", time.Now().Add(-admissionTTL).UnixNano())

	results, err := s.client.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     s.admittedKey(eventID),
		ByScore: true,
		Start:   "0",
		Stop:    cutoff,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("getting expired admitted users: %w", err)
	}
	return results, nil
}

// WaitingCount returns the number of users currently waiting.
// Used by the admission worker to decide batch size.
func (s *QueueStore) WaitingCount(ctx context.Context, eventID string) (int64, error) {
	count, err := s.client.rdb.ZCard(ctx, s.waitingKey(eventID)).Result()
	if err != nil {
		return 0, fmt.Errorf("getting waiting count: %w", err)
	}
	return count, nil
}
