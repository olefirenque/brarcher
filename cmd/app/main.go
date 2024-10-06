package main

import (
	"brarcher/internal/config"
	"brarcher/internal/postgres"
	httpapp "brarcher/internal/server/http"
	"brarcher/internal/server/http/handlers"
	"brarcher/internal/server/http/ws"
	"brarcher/internal/session"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
)

func main() {
	ctx := context.Background()

	conf := config.New()

	repo, err := postgres.Connect(ctx, conf.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to db: %s", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: conf.RedisAddr})

	sessionStore := session.NewSessionStore(redisClient, fmt.Sprintf("%s:%d", conf.HostName, conf.HTTPPort))

	userServer := handlers.NewUserServer(repo)

	redirectServer := handlers.NewRedirectServer(sessionStore)

	messageWSServer := ws.NewMessageWSServer(ws.MessageWSServerDeps{
		Repo:         repo,
		SessionStore: sessionStore,
	})

	mux := http.NewServeMux()
	if err := httpapp.Listen(mux, httpapp.Servers{
		UserServer:      userServer,
		MessageWSServer: messageWSServer,
		RedirectServer:  redirectServer,
	}, conf.HTTPPort); err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
