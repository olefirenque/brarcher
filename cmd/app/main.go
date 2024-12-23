package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"

	httpapp "brarcher/internal/api/http"
	"brarcher/internal/api/http/handlers"
	"brarcher/internal/api/http/ws"
	"brarcher/internal/config"
	"brarcher/internal/logger"
	"brarcher/internal/postgres"
	"brarcher/internal/session"
)

func main() {
	ctx := context.Background()

	conf := config.New()

	repo, err := postgres.Connect(ctx, conf.PostgresDSN)
	if err != nil {
		logger.Fatalf("failed to connect to db: %s", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: conf.RedisAddr})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatalf("failed to connect to redis: %s", err)
	}

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
		logger.Fatalf("listen: %s", err)
	}
}
