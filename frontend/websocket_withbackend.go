package frontend

import (
	"encoding/gob"
	"fmt"
	"net"

	"github.com/demskie/hyperionfive/frontendmessage"

	"github.com/demskie/hyperionfive/backend"
	"github.com/gorilla/websocket"
)

func (s *service) receiveBackendSync(conn *websocket.Conn) error {
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

	var mutated bool
	s.RLock()
	val, exists := s.allServers[inMsg.GalaxyID]
	if !exists {
		mutated = true
	} else if val.Hostname != inMsg.Hostname ||
		val.GlobalX != inMsg.GlobalX ||
		val.GlobalY != inMsg.GlobalY {
		mutated = true
	}
	// BUG: does not account for removal of galaxies
	s.RUnlock()

	s.Lock()
	if mutated {
		s.version++
	}
	s.allServers[inMsg.GalaxyID] = &frontendmessage.Backend{
		Hostname:       inMsg.Hostname,
		GlobalX:        inMsg.GlobalX,
		GlobalY:        inMsg.GlobalY,
		PlayerStrength: inMsg.PlayerStrength,
	}
	s.dynamicUniverse.Reset()
	s.dynamicUniverse.Marshal(&frontendmessage.ServerInfo{
		InstanceName: s.instanceName,
		Version:      s.version,
		Servers:      s.allServers,
	})
	s.Unlock()

	s.RLock()
	outMsg := backend.ToBackendFromFrontend{
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

	err = gob.NewEncoder(connWriter).Encode(&outMsg)
	if err != nil {
		return err
	}
	connWriter.Close()

	return nil
}
