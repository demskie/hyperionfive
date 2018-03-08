package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func kickoffBackendToFrontendCommunication(conn *websocket.Conn, galaxy *GalaxyData) {
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
				if checkError(err, fName, "error receiving message") {
					backoutOnce.Do(killFunc)
					return
				}
				if msgType != websocket.BinaryMessage {
					logInfo(fName, "messageType is invalid")
					return
				}
				networkIn.Reset()
				networkIn.Write(newData)
				inMsg := frontendToBackendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}
				GLOBAL.galaxyInfo.Lock()
				GLOBAL.galaxyInfo.GalaxyName = inMsg.GalaxyName
				GLOBAL.galaxyInfo.GlobalX = inMsg.GlobalX
				GLOBAL.galaxyInfo.GlobalY = inMsg.GlobalY
				GLOBAL.galaxyInfo.Unlock()
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
				GLOBAL.galaxyInfo.RLock()
				outMsg := backendToFrontendMessage{
					GalaxyName: GLOBAL.galaxyInfo.GalaxyName,
					GlobalX:    GLOBAL.galaxyInfo.GlobalX,
					GlobalY:    GLOBAL.galaxyInfo.GlobalY,
					ClientIDs:  make([]string, len(GLOBAL.galaxyInfo.PlayerPtrs)),
				}
				i := 0
				for playerID := range GLOBAL.galaxyInfo.PlayerPtrs {
					outMsg.ClientIDs[i] = playerID
					i++
				}
				GLOBAL.galaxyInfo.RUnlock()
				logDebug(fName, remoteIP, fmt.Sprintf("outMsg= %+v", outMsg))
				// send outgoing message
				networkOut.Reset()
				err := enc.Encode(outMsg)
				if checkError(err, fName, "error encoding outgoing message") {
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
