package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	httpapp "brarcher/internal/api/http/handlers"
	"brarcher/internal/logger"
)

func (ms *MessageWSServer) redirectToTargetSession(ctx context.Context, toID int64, msg string) {
	// if session is handled by current backend, just handle it via message channel
	if ch, ok := ms.sessionStore.GetSessionChan(toID); ok {
		select {
		case <-ctx.Done():
		case ch <- msg:
		}

		return
	}

	// session is handled by another backend, do redirect message
	host, ok := ms.sessionStore.ResolveBackend(ctx, toID)
	if !ok {
		// user isn't connected currently.
		return
	}

	msgRedirect := httpapp.MessageRedirect{
		Message:  msg,
		ToUserID: toID,
	}

	payload, err := json.Marshal(msgRedirect)
	if err != nil {
		logger.Errorf("failed to marshal redirect message: %s", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s/internal/redirect", host), bytes.NewReader(payload))
	if err != nil {
		logger.Errorf("failed to create redirect request: %s", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("failed to send redirect request: %s", err)
		return
	} else if resp.StatusCode != http.StatusOK {
		logger.Errorf("failed to redirect message, got status %s", resp.Status)
		return
	}

	logger.Infof("successfully redirected message to backend: %s", host)
}
