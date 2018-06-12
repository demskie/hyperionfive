package backend

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type frontendToBackendMessage struct {
	GalaxyNameUint uint64
	GlobalX        int64
	GlobalY        int64
	PlayersToSpawn []playerInfo
}

type backendToFrontendMessage struct {
	GalaxyNameUint uint64
	GlobalX        int64
	GlobalY        int64
	ClientIDs      []string
}

// KickoffBackendToFrontendCommunication ...
func (universe *UniverseStruct) KickoffBackendToFrontendCommunication(conn *websocket.Conn, galaxy *GalaxyStruct) {
	const fName = "(backendToFrontend)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// create channel to backout of nested gofuncs and for loops
	////////////////////////////////////////////////////////////

	backoutSignal := make(chan struct{})
	var backoutOnce sync.Once
	killFunc := func() {
		close(backoutSignal)
	}
	defer backoutOnce.Do(killFunc)

	////////////////////////////////////////////////////////////
	// reused objects
	////////////////////////////////////////////////////////////

	var networkIn bytes.Buffer
	var networkOut bytes.Buffer
	dec := gob.NewDecoder(&networkIn)
	enc := gob.NewEncoder(&networkOut)

	////////////////////////////////////////////////////////////
	// receive data from frontend
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				// receive incoming message
				msgType, newData, err := conn.ReadMessage()
				if log.CheckError(err, fName, "error receiving message") {
					backoutOnce.Do(killFunc)
					return
				}
				if msgType != websocket.BinaryMessage {
					log.Info(fName, "messageType is invalid")
					return
				}
				networkIn.Reset()
				networkIn.Write(newData)
				inMsg := frontendToBackendMessage{}
				err = dec.Decode(&inMsg)
				if log.CheckError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}
				universe.Lock()
				universe.GalaxyStruct.Protobuf.NameUint = inMsg.GalaxyNameUint
				universe.GalaxyStruct.Protobuf.GlobalX = inMsg.GlobalX
				universe.GalaxyStruct.Protobuf.GlobalY = inMsg.GlobalY
				universe.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// send data to frontend
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				// prepare outgoing message
				universe.RLock()
				outMsg := backendToFrontendMessage{
					GalaxyNameUint: universe.GalaxyStruct.Protobuf.NameUint,
					GlobalX:        universe.GalaxyStruct.Protobuf.GlobalX,
					GlobalY:        universe.GalaxyStruct.Protobuf.GlobalY,
				}
				//	i := 0
				//	for playerID := range universe.PlayerPtrs {
				//		outMsg.ClientIDs[i] = universe.PlayerNameData.GetString(playerID)
				//		i++
				//	}
				universe.RUnlock()
				log.Debug(fName, remoteIP, fmt.Sprintf("outMsg= %+v", outMsg))
				// send outgoing message
				networkOut.Reset()
				err := enc.Encode(outMsg)
				if log.CheckError(err, fName, "error encoding outgoing message") {
					backoutOnce.Do(killFunc)
					return
				}
				conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())
				time.Sleep(1000 * time.Millisecond)
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}
