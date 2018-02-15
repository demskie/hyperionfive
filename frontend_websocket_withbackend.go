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
	defer GLOBAL.frontendData.RUnlock()
	for galName, backends := range GLOBAL.frontendData.GalBackends {
		for _, backend := range backends {
			if backend == remoteIP {
				return galName
			}
		}
	}
	return ""
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
				galName := getGalName(remoteIP)
				if galName == "" {
					time.Sleep(100 * time.Millisecond)
					continue
				}
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
				time.Sleep(1000 * time.Millisecond * time.Duration(GLOBAL.debugMultiplier))
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

				// check if this server is generating a new galaxy
				if inMsg.GalaxyName == "" && inMsg.Message == "GENERATEGALAXY" {
					logInfo(fName, remoteIP, "is requesting galaxy generation")
					if generateGlobalGalaxyInfo(remoteIP) != true {
						continue
					}
				}

				// no need to continue until other server discovers it's galName
				if inMsg.GalaxyName == "" {
					continue
				}
				galName := getGalName(remoteIP)

				GLOBAL.frontendData.Lock()

				if galName == "" {
					_, ok := GLOBAL.frontendData.GalBackends[inMsg.GalaxyName]
					if !ok {
						GLOBAL.frontendData.GalBackends[inMsg.GalaxyName] = []string{remoteIP}
					} else {
						GLOBAL.frontendData.GalBackends[inMsg.GalaxyName] = append(
							GLOBAL.frontendData.GalBackends[inMsg.GalaxyName], remoteIP)
					}
					galName = inMsg.GalaxyName
				}

				// check GlobalX data
				glblX, ok := GLOBAL.frontendData.GalGlobalX[galName]
				if !ok && inMsg.GlobalX != glblX {
					logInfo(fName, "according to the", remoteIP, "backend GlobalX for",
						galName, "has changed from", fmt.Sprintf("%v", glblX), "to",
						fmt.Sprintf("%v", inMsg.GlobalX))
				}

				// check GlobalY data
				glblY, ok := GLOBAL.frontendData.GalGlobalY[galName]
				if !ok && inMsg.GlobalY != glblY {
					logInfo(fName, "according to the", remoteIP, "backend GlobalY for",
						galName, "has changed from", fmt.Sprintf("%v", glblY), "to",
						fmt.Sprintf("%v", inMsg.GlobalY))
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
