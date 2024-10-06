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

	connMsgChan   map[int64]chan string
	connMsgChanRW sync.RWMutex

	hostName string
}

func NewSessionStore(client *redis.Client, hostname string) *Store {
	return &Store{
		redisClient:   client,
		connMsgChan:   map[int64]chan string{},
		connMsgChanRW: sync.RWMutex{},
		hostName:      hostname,
	}
}

func buildSessionKey(userID int64) string {
	return fmt.Sprintf("session_user_%d", userID)
}

func (ss *Store) StoreSession(ctx context.Context, userID int64) error {
	ss.connMsgChanRW.Lock()
	ss.connMsgChan[userID] = make(chan string, 10)
	ss.connMsgChanRW.Unlock()

	_, err := ss.redisClient.Set(ctx, buildSessionKey(userID), ss.hostName, 5*time.Minute).Result()
	if err != nil {
		return err
	}

	return nil
}

func (ss *Store) DeleteSession(ctx context.Context, userID int64) error {
	if _, err := ss.redisClient.Del(ctx, buildSessionKey(userID)).Result(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (ss *Store) GetSessionChan(userID int64) (chan string, bool) {
	ss.connMsgChanRW.RLock()
	ch, ok := ss.connMsgChan[userID]
	ss.connMsgChanRW.RUnlock()
	return ch, ok
}

func (ss *Store) ResolveBackend(ctx context.Context, userID int64) (string, bool) {
	host, err := ss.redisClient.Get(ctx, buildSessionKey(userID)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			fmt.Printf("unexpected error while resolving backend for user %d: %s", userID, err)
		}
		return "", false
	}

	return host, true
}

func (ss *Store) GetCurrentHost() string {
	return ss.hostName
}
