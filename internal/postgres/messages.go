package postgres

import (
	"context"
	"github.com/georgysavva/scany/pgxscan"
	"time"
)

type Message struct {
	Id      int64  `json:"id"`
	FromID  int64  `json:"from_id"`
	ToID    int64  `json:"to_id"`
	Message string `json:"message"`
}

type roMessageRepository struct {
	query querier
}

type rwMessageRepository struct {
	ROMessages
	exec executor
}

func (ro *roMessageRepository) ListMessages(ctx context.Context, fromID, toID int64, since time.Time) ([]Message, error) {
	const queryName = "MessageRepository/ListMessages"
	const query = `
		select * 
		from messages
		where from_id=$1 and to_id=$2 and stored_at > $3`

	var msgs []Message
	if err := pgxscan.Select(ctx, ro.query, &msgs, query, fromID, toID, since); err != nil {
		return msgs, formatError(queryName, err)
	}

	return msgs, nil
}

func (rw *rwMessageRepository) CreateMessage(ctx context.Context, fromID, toID int64, text string) (int64, error) {
	const queryName = "MessageRepository/CreateMessage"
	const query = `
		insert into messages(from_id, to_id, message)
		values ($1, $2, $3)
		returning message_id`

	var id int64
	if err := pgxscan.Get(ctx, rw.exec, &id, query, fromID, toID, text); err != nil {
		return 0, formatError(queryName, err)
	}

	return id, nil
}
