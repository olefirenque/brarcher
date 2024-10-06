package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"brarcher/internal/logger"
	"brarcher/internal/postgres"
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

	if storeErr := ms.sessionStore.StoreSession(ctx, fromID); storeErr != nil {
		logger.Errorf("failed to store session for user %d: %s", fromID, storeErr)
	}

	logger.Infof("established connection %d->%d", fromID, toID)

	go func() {
		defer ws.Close()

		ws.SetPongHandler(func(string) error { _ = ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
		ws.SetPingHandler(func(string) error { _ = ws.SetWriteDeadline(time.Now().Add(readWait)); return nil })

		// r.Context is closed after hijack
		sessionCtx, cancel := context.WithCancel(context.Background())
		defer cancel() // terminate write loop
		ws.SetCloseHandler(func(_ int, _ string) error {
			cancel()
			return nil
		})

		go ms.writeLoop(sessionCtx, ws, fromID)
		err = ms.readLoop(sessionCtx, ws, fromID, toID)

		var cErr *websocket.CloseError
		if errors.As(err, &cErr) {
			logger.Infof("session graceful shutdown: %s", err)
		} else if err != nil {
			logger.Errorf("ws conn terminated with unexpected error: %s", err)
		}

		// do not use session ctx, as it will be forgotten after the session ends
		if err = ms.sessionStore.DeleteSession(context.Background(), fromID); err != nil {
			logger.Warnf("failed to delete session for user %d: %s", fromID, err)
		}
	}()
}

type inputMessagePayload struct {
	Message string `json:"message"`
}

type outputMessagePayload struct {
	Message string `json:"message"`
	FromId  int64  `json:"from_id"`
}

func (ms *MessageWSServer) writeLoop(ctx context.Context, conn *websocket.Conn, fromID int64) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	ch, ok := ms.sessionStore.GetSessionChan(fromID)
	if !ok {
		logger.Warnf("message chan is not initialized for incoming connection (user=%s)", fromID)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Warnf("failed to send ping to %d user: %s", fromID, err)
			}
		case msg := <-ch:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			rawMessage, err := json.Marshal(outputMessagePayload{
				Message: msg,
				FromId:  fromID,
			})
			if err != nil {
				logger.Errorf("failed to marshal message %s", err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, rawMessage); err != nil {
				logger.Errorf("failed to send message to %d user: %s", fromID, err)
			}
		}
	}
}

var (
	ackMessageSend  = []byte(fmt.Sprintf(`{"status": "ok"}`))
	failMessageSend = []byte(fmt.Sprintf(`{"status": "not_sent"}`))
)

func (ms *MessageWSServer) readLoop(ctx context.Context, conn *websocket.Conn, fromID, toID int64) error {
	for {
		_ = conn.SetReadDeadline(time.Now().Add(readWait))
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("readloop is terminated: %w", err)
		} else if messageType != websocket.TextMessage {
			logger.Warnf("got unexpected message type: %d", messageType)
			continue
		}

		var msg inputMessagePayload
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Errorf("failed to unmarshal message: %s", err)
			continue
		}

		msgID, err := ms.repo.RWMessages().CreateMessage(ctx, fromID, toID, msg.Message)
		if err != nil {
			logger.Errorf("failed to store message: %s", err)
			if err = conn.WriteMessage(websocket.TextMessage, failMessageSend); err != nil {
				logger.Errorf("failed to send failed message status: %w", err)
			}
			continue
		}
		logger.Infof("successfully saved msg %d", msgID)

		// send message to receiver right after message was successfully saved to provide consistency
		go ms.redirectToTargetSession(ctx, toID, msg.Message)

		if err = conn.WriteMessage(websocket.TextMessage, ackMessageSend); err != nil {
			// TODO: consider session termination
			logger.Errorf("failed to send ok message status: %w", err)
			continue
		}

	}
}
