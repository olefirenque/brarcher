DSN="postgresql://postgres:password@localhost:5431/db"

.goose-up:
	goose -dir migration postgres $(DSN) up

.goose-reset:
	goose -dir migration postgres $(DSN) reset