package ws

import (
	"errors"
	"fmt"
	http "net/http"
	"strconv"
	"time"

	"brarcher/internal/logger"
	"brarcher/internal/postgres"
)

const (
	pingPeriod = 60 * time.Second
	pongWait   = 120 * time.Second
	writeWait  = 60 * time.Second
	readWait   = 120 * time.Second
)

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
		msg := fmt.Sprintf("user %d doesn't exist", userID)
		logger.Error(msg)

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(msg))
		return false
	} else if err != nil {
		msg := fmt.Sprintf("failed to validate userID %d: %s", userID, err)
		logger.Error(msg)

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(msg))
		return false
	}

	return true
}

func (ms *MessageWSServer) validateMessageUpgrade(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	fromID, ok := parseIDFromQuery(w, r, "user")
	if !ok {
		return 0, 0, false
	}

	toID, ok := parseIDFromQuery(w, r, "to")
	if !ok {
		return 0, 0, false
	}

	if ok = ms.validateUserID(w, r, fromID); !ok {
		return 0, 0, false
	}

	if ok = ms.validateUserID(w, r, toID); !ok {
		return 0, 0, false
	}

	return fromID, toID, true
}
