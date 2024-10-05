package main

import (
	"brarcher/internal/meta"
	"brarcher/internal/postgres"
	httpapp "brarcher/internal/server/http"
	"brarcher/internal/server/ws"
	"brarcher/internal/session"
	"context"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
)

func main() {
	ctx := context.Background()

	repo, err := postgres.Connect(ctx, "postgresql://postgres:password@localhost:5431/db")
	if err != nil {
		log.Fatalf("failed to connect to db: %s", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6381",
	})

	backendMeta, err := meta.NewMeta(ctx, redisClient)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore := session.NewSessionStore(redisClient)

	userServer := httpapp.NewUserServer(repo)
	messageWSServer := ws.NewMessageWSServer(ws.MessageWSServerDeps{
		Repo:         repo,
		SessionStore: sessionStore,
		Meta:         backendMeta,
	})

	mux := http.NewServeMux()
	if err := httpapp.Listen(mux, httpapp.Servers{
		UserServer:      userServer,
		MessageWSServer: messageWSServer,
	}); err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
