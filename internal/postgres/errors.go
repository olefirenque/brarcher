package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
)

var (
	// ErrNotFound is returned when an entity wasn't found in the database.
	ErrNotFound = errors.New("entity not found")

	// ErrNotModified is returned when an update didn't actually update anything.
	ErrNotModified = errors.New("no rows modified")

	// ErrAlreadyExists is returned when an insert failed due to some unique constraint being violated.
	ErrAlreadyExists = errors.New("entity already exists")
)

func formatError(queryName string, err error) error {
	return fmt.Errorf("executing %s: %w", queryName, err)
}

func handleError(queryName string, err error) error {
	if err != nil {
		return formatError(queryName, err)
	}

	return nil
}

func notFound(queryName string) error {
	return formatError(queryName, ErrNotFound)
}

func notModified(queryName string) error {
	return formatError(queryName, ErrNotModified)
}

func alreadyExists(queryName string) error {
	return formatError(queryName, ErrAlreadyExists)
}

func errIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolated(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}
