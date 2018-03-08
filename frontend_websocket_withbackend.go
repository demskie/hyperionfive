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
				logDebug(fName, remoteIP, fmt.Sprintf("outMsg= %+v", outMsg))
				// send outgoing message
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
				if checkError(err, fName, "error receiving message") {
					backoutOnce.Do(killFunc)
					return
				}
				if msgType != websocket.BinaryMessage {
					logInfo(fName, "messageType is invalid")
					backoutOnce.Do(killFunc)
					return
				}
				networkIn.Reset()
				networkIn.Write(newData)
				inMsg := backendToFrontendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName, remoteIP, fmt.Sprintf("inMsg=  %+v", inMsg))

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
