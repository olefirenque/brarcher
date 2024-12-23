package httpapp

import (
	"fmt"
	"net/http"

	"brarcher/internal/api/http/handlers"
	"brarcher/internal/api/http/ws"
	"brarcher/internal/logger"
)

type Servers struct {
	*handlers.UserServer
	*ws.MessageWSServer
	*handlers.RedirectServer
}

func Listen(mux *http.ServeMux, srv Servers, port int) error {
	mux.HandleFunc("GET /ws", srv.HandleConnection)
	mux.HandleFunc("POST /user", srv.RegisterUser)
	mux.HandleFunc("GET /user", srv.GetUser)
	mux.HandleFunc("POST /internal/redirect", srv.RedirectMessage)

	addr := fmt.Sprintf(":%d", port)
	logger.Infof("http server started on %s", addr)
	return http.ListenAndServe(addr, mux)
}
