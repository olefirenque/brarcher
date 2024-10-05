package httpapp

import (
	"brarcher/internal/server/ws"
	"log"
	"net/http"
)

type Servers struct {
	*UserServer
	*ws.MessageWSServer
}

func Listen(mux *http.ServeMux, srv Servers) error {
	mux.HandleFunc("GET /ws", srv.HandleConnections)
	mux.HandleFunc("POST /user", srv.registerUser)
	mux.HandleFunc("GET /user", srv.getUser)

	log.Println("http server started on :8000")
	return http.ListenAndServe(":8000", mux)
}
