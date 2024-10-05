package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Store struct {
	redisClient *redis.Client

	connPool   map[int64]chan string
	connPoolRW sync.RWMutex
}

func NewSessionStore(client *redis.Client) *Store {
	return &Store{
		redisClient: client,
		connPool:    map[int64]chan string{},
		connPoolRW:  sync.RWMutex{},
	}
}

func buildSessionKey(userID int64) string {
	return fmt.Sprintf("session_user_%d", userID)
}

func (ss *Store) StoreSession(ctx context.Context, userID int64, backendID string) error {
	ss.redisClient.Set(ctx, buildSessionKey(userID), backendID, 5*time.Minute)

	ss.connPoolRW.Lock()
	ss.connPool[userID] = make(chan string, 10)
	ss.connPoolRW.Unlock()

	return nil
}

func (ss *Store) DeleteSession(ctx context.Context, userID int64) error {
	if _, err := ss.redisClient.Del(ctx, buildSessionKey(userID)).Result(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (ss *Store) GetSessionBackendID(ctx context.Context, userID int64) (string, bool, error) {
	backendID, err := ss.redisClient.Get(ctx, buildSessionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	} else if err != nil {
		return "", false, fmt.Errorf("failed to get session: %w", err)
	}
	return backendID, true, nil
}

func (ss *Store) GetSessionChan(userID int64) (chan string, bool) {
	ss.connPoolRW.RLock()
	ch, ok := ss.connPool[userID]
	ss.connPoolRW.RUnlock()
	return ch, ok
}
