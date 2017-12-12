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

func getGalName(remoteIP string) string {
	GLOBAL.frontendData.RLock()
	for galName, backends := range GLOBAL.frontendData.GalBackends {
		for _, backend := range backends {
			if backend == remoteIP {
				GLOBAL.frontendData.RUnlock()
				return galName
			}
		}
	}
	GLOBAL.frontendData.RUnlock()
	return ""
}

func handleBackendToFrontendCommunication(conn *websocket.Conn) {
	const fName = "(backendToFrontend)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// kickoff galaxyinfo generation if necessary
	////////////////////////////////////////////////////////////

	if getGalName(remoteIP) == "" {
		if generateGlobalGalaxyInfo(remoteIP) != true {
			return
		}
	}

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
				galName := getGalName(remoteIP)
				outMsg := frontendToBackendMessage{}
				GLOBAL.frontendData.RLock()
				outMsg.GalaxyName = galName
				outMsg.GlobalX = GLOBAL.frontendData.GalGlobalX[galName]
				outMsg.GlobalY = GLOBAL.frontendData.GalGlobalY[galName]
				outMsg.OtherFrontends = append(outMsg.OtherFrontends,
					GLOBAL.frontendData.OtherFrontends...)
				for _, bcknd := range GLOBAL.frontendData.GalBackends[galName] {
					if bcknd != remoteIP {
						outMsg.OtherBackends = append(outMsg.OtherBackends, bcknd)
					}
				}
				GLOBAL.frontendData.RUnlock()
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

				// handle GalaxyName data
				if inMsg.GalaxyName == "" {
					logInfo(fName, remoteIP, "hasn't been assigned a galaxy yet")
					continue
				}
				galName := getGalName(remoteIP)
				if galName == "" && inMsg.GalaxyName != "" {
					logInfo(fName, remoteIP, "knows about a galaxy that we do not:",
						inMsg.GalaxyName)
					continue
				}
				if inMsg.GalaxyName != galName && galName != "" {
					logError(fName, remoteIP, "is serving the",
						inMsg.GalaxyName, "galaxy instead of", galName)
					continue
				}

				GLOBAL.frontendData.Lock()

				// handle GlobalX data
				glblX, ok := GLOBAL.frontendData.GalGlobalX[galName]
				if !ok && inMsg.GlobalX != glblX {
					logError(fName, "according to", remoteIP, "GlobalX for", galName,
						"has changed from", fmt.Sprintf("%v", glblX), "to",
						fmt.Sprintf("%v", inMsg.GlobalX))
				} else {
					GLOBAL.frontendData.GalGlobalX[galName] = inMsg.GlobalX
				}

				// handle GlobalY data
				glblY, ok := GLOBAL.frontendData.GalGlobalY[galName]
				if !ok && inMsg.GlobalY != glblY {
					logError(fName, "according to", remoteIP, "GlobalY for", galName,
						"has changed from", fmt.Sprintf("%v", glblY), "to",
						fmt.Sprintf("%v", inMsg.GlobalY))
				} else {
					GLOBAL.frontendData.GalGlobalY[galName] = inMsg.GlobalY
				}

				// handle ClientIDs data
				GLOBAL.frontendData.GalClientIDs[galName] = inMsg.ClientIDs

				// handle OtherBackends data
				localBackends, ok := GLOBAL.frontendData.GalBackends[galName]
				if !ok {
					GLOBAL.frontendData.GalBackends[galName] = inMsg.OtherBackends
				} else {
					for _, remoteBknd := range inMsg.OtherBackends {
						foundMatch := false
						for _, localBknd := range localBackends {
							if remoteBknd == localBknd {
								foundMatch = true
								break
							}
						}
						if foundMatch == false {
							localBackends = append(localBackends, remoteBknd)
						}
					}
				}

				// handle OtherFrontends data
				localFrontends := GLOBAL.frontendData.OtherFrontends
				for _, remoteFrnt := range inMsg.OtherFrontends {
					foundMatch := false
					for _, localFrnt := range localFrontends {
						if remoteFrnt == localFrnt {
							foundMatch = true
							break
						}
					}
					if foundMatch == false {
						localFrontends = append(localFrontends, remoteFrnt)
					}
				}

				GLOBAL.frontendData.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}
