-- +goose Up
-- +goose StatementBegin
create table users
(
    id       bigserial primary key,
    username text unique
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
truncate users;
drop table users;
-- +goose StatementEnd
