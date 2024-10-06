package httpapp

import (
	"brarcher/internal/server/http/handlers"
	"brarcher/internal/server/http/ws"
	"fmt"
	"log"
	"net/http"
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
	log.Printf("http server started on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
