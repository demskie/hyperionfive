package backend

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// ToFrontendFromBackend ...
type ToFrontendFromBackend struct {
	GalaxyNameInt uint64
	GlobalX       int64
	GlobalY       int64
	Players       []string
}

// FromBackendToFrontend ...
type FromBackendToFrontend struct {
	PlayersToSpawn map[string]int32
}

func (universe *UniverseStruct) startFrontendSync(conn *websocket.Conn) error {
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	universe.RLock()
	outMsg := ToFrontendFromBackend{
		GalaxyNameInt: universe.GalaxyStruct.Protobuf.NameUint,
		GlobalX:       universe.GalaxyStruct.Protobuf.GlobalX,
		GlobalY:       universe.GalaxyStruct.Protobuf.GlobalY,
		Players:       make([]string, 128), // TODO
	}
	universe.RUnlock()

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

	msgType, connReader, err := conn.NextReader()
	if msgType != websocket.BinaryMessage {
		return fmt.Errorf("%v tried to send a non binary message", remoteIP)
	} else if err != nil {
		return err
	}
	inMsg := FromBackendToFrontend{}
	err = gob.NewDecoder(connReader).Decode(&inMsg)
	if err != nil {
		return err
	}

	return nil
}
