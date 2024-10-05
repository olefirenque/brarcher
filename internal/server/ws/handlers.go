package ws

import (
	"brarcher/internal/meta"
	"brarcher/internal/postgres"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type sessionStore interface {
	StoreSession(ctx context.Context, userID int64, backendID string) error
	DeleteSession(ctx context.Context, userID int64) error
	GetSessionBackendID(ctx context.Context, userID int64) (string, bool, error)
	GetSessionChan(userID int64) (chan string, bool)
}

type MessageWSServer struct {
	upGrader     websocket.Upgrader
	repo         postgres.RepositoryProvider
	sessionStore sessionStore
	meta         *meta.Meta
}

type MessageWSServerDeps struct {
	Repo         postgres.RepositoryProvider
	SessionStore sessionStore
	Meta         *meta.Meta
}

func NewMessageWSServer(deps MessageWSServerDeps) *MessageWSServer {
	return &MessageWSServer{
		upGrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		repo:         deps.Repo,
		meta:         deps.Meta,
		sessionStore: deps.SessionStore,
	}
}

func parseIDFromQuery(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	idRaw := r.URL.Query().Get(key)

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to get `%s` idRaw: %s", key, err)))
		return 0, false
	}

	return id, true
}

func (ms *MessageWSServer) validateUserID(w http.ResponseWriter, r *http.Request, userID int64) bool {
	_, err := ms.repo.ROUsers().GetUser(r.Context(), userID)
	if errors.Is(err, postgres.ErrNotFound) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("user %d doesn't exist", userID)))
		return false
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to validate userID %d: %s", userID, err)))
		return false
	}

	return true
}

func (ms *MessageWSServer) validateMessageUpgrade(w http.ResponseWriter, r *http.Request) (int64, int64) {
	fromID, ok := parseIDFromQuery(w, r, "user")
	if !ok {
		return 0, 0
	}

	toID, ok := parseIDFromQuery(w, r, "to")
	if !ok {
		return 0, 0
	}

	if ok = ms.validateUserID(w, r, fromID); !ok {
		return 0, 0
	}

	if ok = ms.validateUserID(w, r, toID); !ok {
		return 0, 0
	}

	return fromID, toID
}

func (ms *MessageWSServer) HandleConnections(w http.ResponseWriter, r *http.Request) {
	fromID, toID := ms.validateMessageUpgrade(w, r)

	ws, err := ms.upGrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusUpgradeRequired)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to upgrade conn: %s", err)))
		return
	}
	defer ws.Close()

	if sessionErr := ms.sessionStore.StoreSession(r.Context(), fromID, ms.meta.BackendID); sessionErr != nil {
		fmt.Printf("failed to store session from user %d: %s\n", fromID, sessionErr)
	}

	go ms.writeLoop(r.Context(), ws, fromID)

	err = ms.readLoop(r.Context(), ws, fromID, toID)
	var cErr *websocket.CloseError
	if errors.As(err, &cErr) {
		log.Printf("session graceful shutdown: %s\n", err)
	} else if err != nil {
		log.Println(err)
	}
}

type messagePayload struct {
	Message string `json:"message"`
}

func (ms *MessageWSServer) writeLoop(ctx context.Context, conn *websocket.Conn, fromID int64) {
	if ch, ok := ms.sessionStore.GetSessionChan(fromID); ok {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				if err := conn.WriteMessage(1, []byte(msg)); err != nil {
					fmt.Printf("failed to send message to %d user: %s\n", fromID, err)
				}
			}
		}
	}
}

func (ms *MessageWSServer) redirectToTargetSession(ctx context.Context, toID int64, msg string) {
	backendID, exists, err := ms.sessionStore.GetSessionBackendID(ctx, toID)
	if err != nil {
		fmt.Printf("failed to list target user session: %s\n", err)
		return
	} else if !exists {
		return
	}

	if backendID == ms.meta.BackendID {
		fmt.Printf("got message for current backend: %s\n", backendID)
		if ch, ok := ms.sessionStore.GetSessionChan(toID); ok {
			select {
			case <-ctx.Done():
			case ch <- msg:
			}
		}
		return
	}

	// TODO: redirect to another host via stream and write to target user
	fmt.Printf("got message for another backend: %s\n", backendID)
}

func (ms *MessageWSServer) readLoop(ctx context.Context, conn *websocket.Conn, fromID, toID int64) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var msg messagePayload
		if err := json.Unmarshal(message, &msg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		go ms.redirectToTargetSession(ctx, toID, msg.Message)

		msgID, err := ms.repo.RWMessages().CreateMessage(ctx, fromID, toID, msg.Message)
		if err != nil {
			return fmt.Errorf("failed to store message: %w", err)
		}

		fmt.Printf("succesfully saved msg %d\n", msgID)
	}
}
