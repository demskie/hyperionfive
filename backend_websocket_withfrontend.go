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

func kickoffBackendToFrontendCommunication(conn *websocket.Conn) {
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
				logDebug(fName, remoteIP, fmt.Sprintf("inMsg=  %+v", inMsg))

				GLOBAL.backendData.Lock()
				if GLOBAL.backendData.GalaxyName != inMsg.GalaxyName {
					if GLOBAL.backendData.GalaxyName == "" {
						logInfo(fName, remoteIP, "is assigning us galaxyName",
							inMsg.GalaxyName)
					} else {
						logError(fName, "according to", remoteIP, "galaxyName has changed from",
							GLOBAL.backendData.GalaxyName, "to", inMsg.GalaxyName)
					}
					GLOBAL.backendData.GalaxyName = inMsg.GalaxyName
				}
				if GLOBAL.backendData.GlobalX != inMsg.GlobalX {
					if GLOBAL.backendData.GlobalX == 0 {
						logInfo(fName, remoteIP, "is assigning us globalX",
							fmt.Sprintf("%v", inMsg.GlobalX))
					} else {
						logError(fName, "according to", remoteIP, "globalX has changed from",
							fmt.Sprintf("%v", GLOBAL.backendData.GlobalX), "to",
							fmt.Sprintf("%v", inMsg.GlobalX))
					}
					GLOBAL.backendData.GlobalX = inMsg.GlobalX
				}
				if GLOBAL.backendData.GlobalY != inMsg.GlobalY {
					if GLOBAL.backendData.GlobalY == 0 {
						logInfo(fName, remoteIP, "is assigning us globalY",
							fmt.Sprintf("%v", inMsg.GlobalY))
					} else {
						logError(fName, "according to", remoteIP, "globalY has changed from",
							fmt.Sprintf("%v", GLOBAL.backendData.GlobalY), "to",
							fmt.Sprintf("%v", inMsg.GlobalY))
					}
					GLOBAL.backendData.GlobalY = inMsg.GlobalY
				}
				localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
				for _, remoteBackend := range inMsg.OtherBackends {
					if remoteBackend == localIP {
						continue
					}
					GLOBAL.backendData.Backends = appendStringIfMissing(GLOBAL.backendData.Backends,
						remoteBackend)
				}
				for _, remoteFrontend := range inMsg.OtherFrontends {
					GLOBAL.backendData.Frontends = appendStringIfMissing(GLOBAL.backendData.Frontends,
						remoteFrontend)
				}
				GLOBAL.backendData.Unlock()
			}
		}
	}()

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
				outMsg := backendToFrontendMessage{}
				GLOBAL.backendData.RLock()
				outMsg.GalaxyName = GLOBAL.backendData.GalaxyName
				outMsg.GlobalX = GLOBAL.backendData.GlobalX
				outMsg.GlobalY = GLOBAL.backendData.GlobalY
				for _, ids := range GLOBAL.backendData.ClientIDs {
					outMsg.ClientIDs = append(outMsg.ClientIDs, ids...)
				}
				outMsg.OtherBackends = append(outMsg.OtherBackends,
					GLOBAL.backendData.Backends...)
				outMsg.OtherFrontends = append(outMsg.OtherFrontends,
					GLOBAL.backendData.Frontends...)
				GLOBAL.backendData.RUnlock()
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
