package frontend

import (
	"time"

	"github.com/gorilla/websocket"
)

type toFrontendMessage struct {
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends map[string]time.Time
}

func (s *service) receiveFrontendSync(conn *websocket.Conn) error {
	const fName = "(frontendToFrontend)"

	return nil
}

func (s *service) startFrontendSync(conn *websocket.Conn) error {
	const fName = "(frontendToFrontend)"

	return nil
}
