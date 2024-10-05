package postgres

import (
	"context"
	"github.com/georgysavva/scany/pgxscan"
)

type User struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

type roUserRepository struct {
	query querier
}

type rwUserRepository struct {
	ROUsers
	exec executor
}

func (ro *roUserRepository) GetUser(ctx context.Context, userID int64) (User, error) {
	const queryName = "UserRepository/GetUser"
	const query = `
		select * 
		from users
		where id = $1`

	var user User
	if err := pgxscan.Get(ctx, ro.query, &user, query, userID); errIsNoRows(err) {
		return user, notFound(queryName)
	} else if err != nil {
		return user, formatError(queryName, err)
	}

	return user, nil
}

func (rw *rwUserRepository) CreateUser(ctx context.Context, username string) (int64, error) {
	const queryName = "UserRepository/CreateUser"
	const query = `
		insert into users(username)
		values ($1)
		returning id`

	var id int64
	if err := pgxscan.Get(ctx, rw.exec, &id, query, username); isUniqueViolated(err) {
		return 0, alreadyExists(queryName)
	} else if err != nil {
		return 0, formatError(queryName, err)
	}

	return id, nil
}
