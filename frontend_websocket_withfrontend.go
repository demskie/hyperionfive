package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func frontendToFrontendCommunication(conn *websocket.Conn) {
	const fName = "(frontendToFrontend)"
	//remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

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
	// receive data from other frontend server
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
				inMsg := frontendToFrontendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName, fmt.Sprint("inMsg=\t\t", inMsg))

				GLOBAL.frontendData.Lock()
				GLOBAL.frontendData.GalGlobalX = inMsg.GalGlobalX
				GLOBAL.frontendData.GalGlobalY = inMsg.GalGlobalY
				GLOBAL.frontendData.GalClientIDs = inMsg.GalClientIDs
				GLOBAL.frontendData.GalBackends = inMsg.GalBackends
				GLOBAL.frontendData.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// send data to other frontend server
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				// prepare outgoing message
				outMsg := frontendToFrontendMessage{}
				GLOBAL.frontendData.RLock()
				for key, val := range GLOBAL.frontendData.GalGlobalX {
					outMsg.GalGlobalX[key] = val
				}
				for key, val := range GLOBAL.frontendData.GalGlobalY {
					outMsg.GalGlobalY[key] = val
				}
				for key, val := range GLOBAL.frontendData.GalClientIDs {
					outMsg.GalClientIDs[key] = append(outMsg.GalClientIDs[key], val...)
				}
				for key, val := range GLOBAL.frontendData.GalBackends {
					outMsg.GalBackends[key] = append(outMsg.GalBackends[key], val...)
				}
				outMsg.OtherFrontends = append(outMsg.OtherFrontends,
					GLOBAL.frontendData.OtherFrontends...)
				GLOBAL.frontendData.RUnlock()
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
