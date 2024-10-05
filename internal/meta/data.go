package meta

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	backendIDLenght = 10
	leaseTime       = 10 * time.Minute
)

type Meta struct {
	redisClient *redis.Client

	BackendID string
}

func NewMeta(ctx context.Context, client *redis.Client) (*Meta, error) {
	m := &Meta{redisClient: client}
	if err := m.AcquireBackendID(ctx); err != nil {
		return nil, fmt.Errorf("failed to initially acquire id: %w", err)
	}

	go m.extendBackendIDLease(ctx)
	return m, nil
}

var errBackendIDAlreadyAcquired = errors.New("backendID is already acquired")

func (m *Meta) AcquireBackendID(ctx context.Context) error {
	for {
		backendID := fmt.Sprintf("backend-%s", randSeq(backendIDLenght))

		txf := func(tx *redis.Tx) error {
			if value, err := tx.Get(ctx, backendID).Bool(); err != nil && !errors.Is(err, redis.Nil) {
				return fmt.Errorf("failed to check backendID: %w", err)
			} else if value {
				return errBackendIDAlreadyAcquired
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, backendID, "true", leaseTime)
				return nil
			})
			return err
		}

		err := m.redisClient.Watch(ctx, txf, backendID)
		if errors.Is(err, errBackendIDAlreadyAcquired) {
			continue
		} else if err != nil {
			return err
		}

		m.BackendID = backendID
		return nil
	}
}

func extendLeaseQuery(extendDuration time.Duration) string {
	return fmt.Sprintf(
		`eval 'local ttl = redis.call("TTL", KEYS[1]) + %d; redis.call("EXPIRE", KEYS[1], ttl)' 0`,
		extendDuration/time.Second,
	)
}

func (m *Meta) extendBackendIDLease(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(leaseTime / 3 * 2):
			if _, err := m.redisClient.Eval(ctx, extendLeaseQuery(leaseTime), []string{m.BackendID}).Result(); err != nil {
				fmt.Printf("failed to lease id %s: %s\n", m.BackendID, err)
			}
		}
	}
}
