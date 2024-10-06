package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"brarcher/internal/postgres"
	"github.com/gorilla/websocket"
)

type sessionStore interface {
	StoreSession(ctx context.Context, userID int64) error
	DeleteSession(ctx context.Context, userID int64) error
	GetSessionChan(userID int64) (chan string, bool)
	ResolveBackend(ctx context.Context, userID int64) (string, bool)
}

type MessageWSServer struct {
	upGrader     websocket.Upgrader
	repo         postgres.RepositoryProvider
	sessionStore sessionStore
}

type MessageWSServerDeps struct {
	Repo         postgres.RepositoryProvider
	SessionStore sessionStore
}

func NewMessageWSServer(deps MessageWSServerDeps) *MessageWSServer {
	return &MessageWSServer{
		upGrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		repo:         deps.Repo,
		sessionStore: deps.SessionStore,
	}
}

func (ms *MessageWSServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fromID, toID, ok := ms.validateMessageUpgrade(w, r)
	if !ok {
		return
	}

	ws, err := ms.upGrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusUpgradeRequired)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to upgrade conn: %s", err)))
		return
	}
	defer ws.Close()

	if storeErr := ms.sessionStore.StoreSession(ctx, fromID); storeErr != nil {
		fmt.Printf("failed to store session for user %d: %s\n", fromID, storeErr)
	}

	go ms.writeLoop(ctx, ws, fromID)
	err = ms.readLoop(ctx, ws, fromID, toID)
	var cErr *websocket.CloseError
	if errors.As(err, &cErr) {
		log.Printf("session graceful shutdown: %s\n", err)
	} else if err != nil {
		log.Println(err)
	}

	if err = ms.sessionStore.DeleteSession(ctx, fromID); err != nil {
		fmt.Printf("failed to delete session for user %d: %s\n", fromID, err)
	}
}

type messagePayload struct {
	Message string `json:"message"`
}

func (ms *MessageWSServer) writeLoop(ctx context.Context, conn *websocket.Conn, fromID int64) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	ch, ok := ms.sessionStore.GetSessionChan(fromID)
	if !ok {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Printf("failed to send ping to %d user: %s\n", fromID, err)
			}
		case msg := <-ch:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				fmt.Printf("failed to send message to %d user: %s\n", fromID, err)
			}
		}
	}
}

func (ms *MessageWSServer) readLoop(ctx context.Context, conn *websocket.Conn, fromID, toID int64) error {
	conn.SetPongHandler(func(string) error { _ = conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_ = conn.SetReadDeadline(time.Now().Add(readWait))
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var msg messagePayload
		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("failed to unmarshal message: %s\n", err)
			continue
		}

		go ms.redirectToTargetSession(ctx, toID, msg.Message)

		msgID, err := ms.repo.RWMessages().CreateMessage(ctx, fromID, toID, msg.Message)
		if err != nil {
			return fmt.Errorf("failed to store message: %w", err)
		}

		fmt.Printf("succesfully saved msg %d\n", msgID)
	}
}
