package frontend

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/demskie/hyperionfive/backend"
	"github.com/gorilla/websocket"
)

func (s *Service) receiveBackendSync(conn *websocket.Conn) error {
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	msgType, connReader, err := conn.NextReader()
	if msgType != websocket.BinaryMessage {
		return fmt.Errorf("%v tried to send a non binary message", remoteIP)
	} else if err != nil {
		return err
	}
	inMsg := backend.ToFrontendFromBackend{}
	err = gob.NewDecoder(connReader).Decode(&inMsg)
	if err != nil {
		return err
	}

	s.RLock()
	outMsg := backend.FromBackendToFrontend{
		PlayersToSpawn: make(map[string]int32, len(s.playersToSpawn)),
	}
	for k, v := range s.playersToSpawn {
		outMsg.PlayersToSpawn[k] = v
	}
	s.RUnlock()

	connWriter, err := conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	defer connWriter.Close()

	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	err = gob.NewEncoder(connWriter).Encode(&outMsg)
	if err != nil {
		return err
	}
	connWriter.Close()
	conn.SetWriteDeadline(time.Time{})

	return nil
}
