package main

import (
	"bytes"
	"encoding/gob"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

func backendToBackendCommunication(conn *websocket.Conn) {
	const fName = "(backendToBackend)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	//localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())

	//
	// TODO // create defer func() for removing data when connection terminates
	//

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
	remoteSystemsInRecovery := []string{}
	remoteSystemsInRecoveryMutex := sync.Mutex{}

	////////////////////////////////////////////////////////////
	// receive data from backend
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				msgType, newData, err := conn.ReadMessage()
				if checkError(err, fName, "error receiving message:") {
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
				dec := gob.NewDecoder(&networkIn)
				inMsg := backendToBackendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}

				// logDebug(fName, remoteIP, fmt.Sprintf("inMsg=  %+v", inMsg))

				// WARNING // do not try to lessen the scope of this lock
				GLOBAL.backendData.Lock()

				// record receipt of message
				GLOBAL.backendData.ReceivedMessageFrom = appendStringIfMissing(
					GLOBAL.backendData.ReceivedMessageFrom, remoteIP)

				// see if we need to trigger the allServersReadySignal
				resetChannel := true
				for _, knownBackend := range GLOBAL.backendData.Backends {
					foundMatch := false
					for _, aliveBackend := range GLOBAL.backendData.ReceivedMessageFrom {
						if knownBackend == aliveBackend {
							foundMatch = true
							break
						}
					}
					if foundMatch == false {
						resetChannel = false
						break
					}
				}
				if resetChannel {
					GLOBAL.backendData.ReceivedMessageFrom = []string{}
					close(GLOBAL.allServersReadySignal)
					GLOBAL.allServersReadySignal = make(chan struct{})
				}

				// add frontend to slice if necessary
				for _, frontend := range GLOBAL.backendData.Frontends {
					GLOBAL.backendData.Frontends = appendStringIfMissing(
						GLOBAL.backendData.Frontends, frontend)
				}

				// add backend to slice if necessary
				for _, backend := range GLOBAL.backendData.Backends {
					GLOBAL.backendData.Backends = appendStringIfMissing(
						GLOBAL.backendData.Backends, backend)
				}

				// update system hashes
				for sysName, sysHash := range inMsg.SystemHashes {
					_, ok := GLOBAL.backendData.NeighborSystemHashes[remoteIP]
					if !ok {
						GLOBAL.backendData.NeighborSystemHashes[remoteIP] = map[string]int64{}
					}
					GLOBAL.backendData.NeighborSystemHashes[remoteIP][sysName] = sysHash
				}

				// merge Buffered requests
				GLOBAL.allRequests.Lock()
				for uniqueID, remoteRequest := range inMsg.BufferedRequests {
					GLOBAL.allRequests.Buffered[uniqueID] = remoteRequest
					// discover and add to ConfirmedIDs
					_, exists := GLOBAL.allRequests.ConfirmedIDs[remoteIP]
					if exists == false {
						GLOBAL.allRequests.ConfirmedIDs[remoteIP] = map[int64]emptyStruct{}
					}
					GLOBAL.allRequests.ConfirmedIDs[remoteIP][uniqueID] = emptyStruct{}
				}
				GLOBAL.allRequests.Unlock()

				// BLINDLY OVERWRITE SystemBytes
				for sysName, sysBytes := range inMsg.SystemBytes {
					_, exists := GLOBAL.backendData.SystemsInRecovery[sysName]
					if exists {
						GLOBAL.backendData.SystemBytes[sysName] = sysBytes
						GLOBAL.backendData.SystemsInRecovery[sysName] = true
					} else {
						logError(fName, remoteIP, "thinks that we have a system in recovery", sysName)
					}
				}

				// reset systemsInRecovery
				remoteSystemsInRecoveryMutex.Lock()
				remoteSystemsInRecovery = []string{} // clear the shared resource
				remoteSystemsInRecovery = append(remoteSystemsInRecovery, inMsg.SystemsInRecovery...)
				remoteSystemsInRecoveryMutex.Unlock()

				// enforce and/or set SystemOrder
				GLOBAL.galaxyMutex.RLock()
				numSystems := len(GLOBAL.galaxy.Solarsystemslice)
				GLOBAL.galaxyMutex.Unlock()
				if numSystems != len(inMsg.SystemOrder) {
					if numSystems == 0 {
						GLOBAL.galaxyMutex.Lock()
						GLOBAL.galaxy.Solarsystemslice = make([]*SolarSystem, len(inMsg.SystemOrder))
						GLOBAL.galaxyMutex.Unlock()
					} else {
						logError(fName, remoteIP, "disagrees with me on the number of systems")
						if remoteIP > getLocalAddress().IP.String() {
							logError(fName, "resetting systems due to disagreement")
							GLOBAL.galaxyMutex.Lock()
							GLOBAL.galaxy.Solarsystemslice = make([]*SolarSystem, len(inMsg.SystemOrder))
							GLOBAL.galaxyMutex.Unlock()
						}
					}
				}
				GLOBAL.galaxyMutex.Unlock()
				// TODO // only do this if remote side has authority (or is more valid)
				GLOBAL.galaxyMutex.RLock()
				for i, remoteSysName := range inMsg.SystemOrder {
					if remoteSysName == "" {
						logError(fName, remoteIP, "is sending empty sysName values")
						continue
					}
					val := GLOBAL.galaxy.Solarsystemslice[i]
					if val == nil {
						val = &SolarSystem{Name: remoteSysName}
					} else if val.Name != "" && val.Name != remoteSysName {
						logError(fName, remoteIP, "disagrees with me on sysName")
					} else {
						val.Name = remoteSysName
					}
				}
				GLOBAL.galaxyMutex.RUnlock()

				// update NeighborTickTimes
				GLOBAL.backendData.NeighborTickTimes[remoteIP] = inMsg.TickTime

				// WARNING // do not try to lessen the scope of this lock
				GLOBAL.backendData.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// send data to backend
	////////////////////////////////////////////////////////////

	go func() {
		for {
			dataReadySignal := make(chan struct{})
			rawData := []byte{}

			<-GLOBAL.backendSignal

			go func() {
				outMsg := backendToBackendMessage{}

				// WARNING // do not try to lessen the scope of this lock
				GLOBAL.backendData.RLock()

				// append Frontends
				outMsg.Frontends = append(outMsg.Frontends,
					GLOBAL.backendData.Frontends...)

				// append Backends
				outMsg.Backends = append(outMsg.Backends,
					GLOBAL.backendData.Backends...)

				// set Priority
				outMsg.Priority = GLOBAL.serverPriority

				// copy SystemHashes
				outMsg.SystemHashes = map[string]int64{}
				for sysName, hash := range GLOBAL.backendData.SystemHashes {
					outMsg.SystemHashes[sysName] = hash
				}

				// copy Buffered requests
				GLOBAL.allRequests.Lock()
				for key, val := range GLOBAL.allRequests.Buffered {
					outMsg.BufferedRequests[key] = val
				}
				GLOBAL.allRequests.Unlock()

				// WARNING // do not try to lessen the scope of this lock
				remoteSystemsInRecoveryMutex.Lock()

				// append SystemBytes
				currentBytes := 0
				maxBytes := 125000000 / 10 / 60          // 1000mbit / 10 / 60ticks
				outMsg.SystemBytes = map[string][]byte{} // TODO // why is this necessary?
				for _, sysName := range remoteSystemsInRecovery {
					outMsg.SystemBytes[sysName] = []byte{}
					outMsg.SystemBytes[sysName] = append(outMsg.SystemBytes[sysName],
						GLOBAL.backendData.SystemBytes[sysName]...)
					currentBytes += len(GLOBAL.backendData.SystemBytes[sysName])
					if currentBytes > maxBytes {
						break
					}
				}

				// send SystemsInRecovery to remote side
				outMsg.SystemsInRecovery = append(outMsg.SystemsInRecovery,
					GLOBAL.backendData.NeighborRecoveryMapping[remoteIP]...)

				// WARNING // do not try to lessen the scope of this lock
				remoteSystemsInRecoveryMutex.Unlock()

				// inform remote server of the system order
				GLOBAL.galaxyMutex.RLock()
				for _, ss := range GLOBAL.galaxy.Solarsystemslice {
					if ss == nil {
						logError(fName, "unitialized SolarSystem detected - not sending SystemOrder")
						break
					}
					outMsg.SystemOrder = append(outMsg.SystemOrder, ss.Name)
				}
				GLOBAL.galaxyMutex.RUnlock()

				// inform remote server how long it took to process the PREVIOUS TICK
				outMsg.TickTime = GLOBAL.backendData.LocalTickTime

				// WARNING // do not try to lessen the scope of this lock
				GLOBAL.backendData.RUnlock()

				// logDebug(fName, remoteIP, fmt.Sprintf("outMsg= %+v", outMsg))

				networkOut.Reset()
				enc := gob.NewEncoder(&networkOut)
				err := enc.Encode(outMsg)
				if checkError(err, fName, "error encoding outgoing message") {
					backoutOnce.Do(killFunc)
					return
				}

				rawData = networkOut.Bytes()

				close(dataReadySignal)
			}()

			select {
			case <-GLOBAL.turningDataSignal:
				logTelemetry(fName, "throwing away outgoing message due to data mutation")
				<-dataReadySignal // wait for above goroutine to complete
			case <-dataReadySignal:
				err := conn.WriteMessage(websocket.BinaryMessage, rawData)
				if checkError(err, fName, "error sending message to client") {
					backoutOnce.Do(killFunc)
					return
				}
			case <-backoutSignal:
				return
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}
