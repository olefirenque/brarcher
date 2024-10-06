package handlers

import (
	"encoding/json"
	"net/http"
)

type sessionStore interface {
	GetSessionChan(userID int64) (chan string, bool)
}

type RedirectServer struct {
	sessionStore sessionStore
}

func NewRedirectServer(sessionStore sessionStore) *RedirectServer {
	return &RedirectServer{sessionStore: sessionStore}
}

type MessageRedirect struct {
	Message  string `json:"message"`
	ToUserID int64  `json:"to_user_id"`
}

func (rs *RedirectServer) RedirectMessage(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var msg MessageRedirect
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("expected valid json body"))
		return
	}

	if ch, ok := rs.sessionStore.GetSessionChan(msg.ToUserID); ok {
		ch <- msg.Message
	}
}
