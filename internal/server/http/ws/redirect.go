package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	httpapp "brarcher/internal/server/http/handlers"
)

func (ms *MessageWSServer) redirectToTargetSession(ctx context.Context, toID int64, msg string) {
	if ch, ok := ms.sessionStore.GetSessionChan(toID); ok {
		select {
		case <-ctx.Done():
		case ch <- msg:
		}

		return
	}

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
		fmt.Printf("failed to marshal redirect message: %s\n", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s/internal/redirect", host), bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("failed to create redirect request: %s\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to send redirect request: %s\n", err)
		return
	} else if resp.StatusCode != http.StatusOK {
		fmt.Printf("failed to redirect message, got status %s\n", resp.Status)
		return
	}

	fmt.Printf("succesfully redirected message to backend: %s\n", host)
}
