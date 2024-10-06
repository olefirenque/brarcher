package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"

	"brarcher/internal/logger"
)

type Store struct {
	redisClient *redis.Client

	connMsgChans   map[int64]chan string
	connMsgChansRW sync.RWMutex

	hostName string
}

func NewSessionStore(client *redis.Client, hostname string) *Store {
	return &Store{
		redisClient:    client,
		connMsgChans:   map[int64]chan string{},
		connMsgChansRW: sync.RWMutex{},
		hostName:       hostname,
	}
}

func buildSessionKey(userID int64) string {
	return fmt.Sprintf("session_user_%d", userID)
}

func (ss *Store) StoreSession(ctx context.Context, userID int64) error {
	ss.connMsgChansRW.Lock()
	ss.connMsgChans[userID] = make(chan string, 10)
	ss.connMsgChansRW.Unlock()

	// Session key must be deleted with a graceful close of a connection, so set long enough ttl.
	// TODO: Consider extending this ttl in the background.
	_, err := ss.redisClient.Set(ctx, buildSessionKey(userID), ss.hostName, time.Hour).Result()
	if err != nil {
		return fmt.Errorf("failed to announce ws session ownership for user %d: %w", userID, err)
	}

	return nil
}

func (ss *Store) DeleteSession(ctx context.Context, userID int64) error {
	ss.connMsgChansRW.Lock()
	delete(ss.connMsgChans, userID)
	ss.connMsgChansRW.Unlock()

	if _, err := ss.redisClient.Del(ctx, buildSessionKey(userID)).Result(); err != nil {
		return fmt.Errorf("failed to forget ws session ownership for user %d: %w", userID, err)
	}
	return nil
}

func (ss *Store) GetSessionChan(userID int64) (chan string, bool) {
	ss.connMsgChansRW.RLock()
	ch, ok := ss.connMsgChans[userID]
	ss.connMsgChansRW.RUnlock()
	return ch, ok
}

func (ss *Store) ResolveBackend(ctx context.Context, userID int64) (string, bool) {
	host, err := ss.redisClient.Get(ctx, buildSessionKey(userID)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			logger.Warnf("unexpected error while resolving backend for user %d: %s", userID, err)
		}
		return "", false
	}

	return host, true
}

func (ss *Store) GetCurrentHost() string {
	return ss.hostName
}
