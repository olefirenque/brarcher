package postgres

import (
	"context"
	"time"
)

var (
	_ DBInterface        = &Database{}
	_ RepositoryProvider = &WriteTx{}
)

// RepositoryProvider is a public interface of a repository layer.
type RepositoryProvider interface {
	ROUsers() ROUsers
	RWUsers() RWUsers
	ROMessages() ROMessages
	RWMessages() RWMessages
}

// DBInterface is a public interface of a database.
type DBInterface interface {
	RepositoryProvider
	RunInTx(ctx context.Context, f func(tx RepositoryProvider) error, isoLevel TxIsoLevel) error
}

// ROUsers is a read-only postgres-based repository for users.
type ROUsers interface {
	GetUser(ctx context.Context, userID int64) (User, error)
}

// RWUsers is a read-write postgres-based repository for users.
type RWUsers interface {
	CreateUser(ctx context.Context, username string) (int64, error)
	ROUsers
}

// ROMessages is a read-only postgres-based repository for messages.
type ROMessages interface {
	ListMessages(ctx context.Context, fromID int64, toID int64, since time.Time) ([]Message, error)
}

// RWMessages is a read-write postgres-based repository for messages.
type RWMessages interface {
	ROMessages
	CreateMessage(ctx context.Context, fromID int64, toID int64, text string) (int64, error)
}
