package frontend

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// frontendToBackendMessage ...
type frontendToBackendMessage struct {
	GalaxyNameInt  int32
	GlobalX        int64
	GlobalY        int64
	PlayersToSpawn []playerInfo
}

// backendToFrontendMessage ...
type backendToFrontendMessage struct {
	GalaxyNameInt int32
	GlobalX       int64
	GlobalY       int64
	ClientIDs     []string
}

type playerInfo struct {
	uniqueID int32
	strength int
}

func handleBackendToFrontendCommunication(conn *websocket.Conn) {
	const fName = "(backendToFrontend)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// kickoff galaxyinfo generation if necessary
	////////////////////////////////////////////////////////////

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
	// resused objects
	////////////////////////////////////////////////////////////

	var networkIn bytes.Buffer
	var networkOut bytes.Buffer
	dec := gob.NewDecoder(&networkIn)
	enc := gob.NewEncoder(&networkOut)

	////////////////////////////////////////////////////////////
	// send data to backend
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				// prepare outgoing message
				networkOut.Reset()
				outMsg := frontendToBackendMessage{
				// TODO // need to reimplement
				}
				log.Debug(fName, remoteIP, fmt.Sprintf("outMsg= %+v", outMsg))
				// send outgoing message
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
	// receive data from backend
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
					backoutOnce.Do(killFunc)
					return
				}
				networkIn.Reset()
				networkIn.Write(newData)
				inMsg := backendToFrontendMessage{}
				err = dec.Decode(&inMsg)
				if log.CheckError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}
				log.Debug(fName, remoteIP, fmt.Sprintf("inMsg=  %+v", inMsg))

				// TODO // need to reimplement
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}
