package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
)

// TxIsoLevel specifies the transaction isolation level.
type TxIsoLevel = pgx.TxIsoLevel

// Transaction isolation levels
const (
	Serializable    = pgx.Serializable
	RepeatableRead  = pgx.RepeatableRead
	ReadCommitted   = pgx.ReadCommitted
	ReadUncommitted = pgx.ReadUncommitted
)

// Database is a high-level pooled connection to a DB cluster.
// Repository methods accessed through Database are run in implicit transactions.
type Database struct {
	pool *pgxpool.Pool
}

// Connect connects to a database cluster using the specified DSN and warden host for DB name resolution.
func Connect(ctx context.Context, dsn string) (*Database, error) {
	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("initializing databasepg cluster: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Close closes all connections to the DB cluster.
func (db *Database) Close() {
	db.pool.Close()
}

// ROUsers returns a user repository which will use this database for querying.
func (db *Database) ROUsers() ROUsers {
	return &roUserRepository{query: clusterReader{db.pool}}
}

// RWUsers returns a user repository which will use this database for querying and executing.
func (db *Database) RWUsers() RWUsers {
	return &rwUserRepository{
		ROUsers: db.ROUsers(),
		exec:    clusterWriter{db.pool},
	}
}

// ROMessages returns a message repository which will use this database for querying.
func (db *Database) ROMessages() ROMessages {
	return &roMessageRepository{query: clusterReader{db.pool}}
}

// RWMessages returns a message repository which will use this database for querying and executing.
func (db *Database) RWMessages() RWMessages {
	return &rwMessageRepository{
		ROMessages: db.ROMessages(),
		exec:       clusterWriter{db.pool},
	}
}

// WriteTx is an active writeable and readable transaction launched by a Database instance.
// Repository methods accessed through WriteTx are run in this transaction.
type WriteTx struct {
	wrapped pgx.Tx
}

// ROUsers returns a user repository which will user this transaction for querying.
func (tx *WriteTx) ROUsers() ROUsers {
	return &roUserRepository{query: tx.wrapped}
}

// RWUsers returns a user repository which will user this transaction for querying and execution.
func (tx *WriteTx) RWUsers() RWUsers {
	return &rwUserRepository{
		ROUsers: tx.ROUsers(),
		exec:    tx.wrapped,
	}
}

func (tx *WriteTx) ROMessages() ROMessages {
	return &roMessageRepository{query: tx.wrapped}
}

func (tx *WriteTx) RWMessages() RWMessages {
	return &rwMessageRepository{
		ROMessages: tx.ROMessages(),
		exec:       tx.wrapped,
	}
}

// RunInTx runs the specified function in a transaction which supports writing and reading.
func (db *Database) RunInTx(ctx context.Context, f func(tx RepositoryProvider) error, isoLevel TxIsoLevel) error {
	return db.pool.BeginTxFunc(ctx,
		pgx.TxOptions{
			IsoLevel: isoLevel,
		},
		func(tx pgx.Tx) error {
			tns := &WriteTx{wrapped: tx}
			return f(tns)
		},
	)
}

// querier can only be used for reading data
type querier = pgxscan.Querier

// executor can be used both for reading and writing data
type executor interface {
	querier
	Exec(ctx context.Context, sql string, args ...any) (commandTag pgconn.CommandTag, err error)
}

// clusterReader implements querier with a read-only cluster connection.
type clusterReader struct {
	pool *pgxpool.Pool
}

func (cr clusterReader) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := cr.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	} else if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("received error from rows: %w", err)
	}

	return rows, nil
}

// clusterWriter implements executor with a read-write cluster connection.
type clusterWriter struct {
	pool *pgxpool.Pool
}

func (cw clusterWriter) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := cw.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	} else if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("received error from rows: %w", err)
	}

	return rows, nil
}

func (cw clusterWriter) Exec(ctx context.Context, sql string, args ...any) (commandTag pgconn.CommandTag, err error) {
	return cw.pool.Exec(ctx, sql, args...)
}
