-- +goose Up
-- +goose StatementBegin
create table messages
(
    message_id bigserial primary key,
    from_id    bigint,
    to_id      bigint,
    message    text,
    stored_at  date default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
truncate table messages;
drop table messages;
-- +goose StatementEnd
