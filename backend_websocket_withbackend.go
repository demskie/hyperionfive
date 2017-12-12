package main

import (
	"bytes"
	"container/ring"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/gorilla/websocket"
)

func backendToBackendCommunication(conn *websocket.Conn) {
	const fName = "(backendToBackend)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())

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
				inMsg := backendToBackendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}

				GLOBAL.backendData.Lock()

				// record receipt of message
				GLOBAL.backendData.ReceivedMessageFrom = appendStringIfMissing(
					GLOBAL.backendData.ReceivedMessageFrom, remoteIP)
				missedMatch := false
				for _, knownBackend := range GLOBAL.backendData.Backends {
					for _, aliveBackend := range GLOBAL.backendData.ReceivedMessageFrom {
						if knownBackend == aliveBackend {
							missedMatch = true
							break
						}
					}
				}
				if missedMatch == false {
					close(GLOBAL.allServersReadySignal)
					//GLOBAL.backendData.ReceivedMessageFrom = []string{"0.0.0.0"} // TODO: remove this
				}

				// update system hashes
				for i := range GLOBAL.backendData.SystemHashes {
					for sysName, localHash := range inMsg.SystemHashes[i] {
						GLOBAL.backendData.SystemHashes[i][remoteIP][sysName] = localHash
					}
				}

				// update priority if not defined
				_, ok := GLOBAL.backendData.Priorities[remoteIP]
				if !ok {
					GLOBAL.backendData.Priorities[remoteIP] = inMsg.Priority
					// initialize empty ring as well
					GLOBAL.backendData.Latencies[remoteIP] = &ring.Ring{}
				} else if inMsg.Priority != GLOBAL.backendData.Priorities[remoteIP] {
					logError(fName, remoteIP, "priority has changed from",
						fmt.Sprintf("%v", inMsg.Priority), "to",
						fmt.Sprintf("%v", GLOBAL.backendData.Priorities[remoteIP]))
					backoutOnce.Do(killFunc)
					return
				}

				// add frontend to slice if necessary
				for _, frontend := range GLOBAL.backendData.Frontends {
					GLOBAL.backendData.Frontends = appendStringIfMissing(GLOBAL.backendData.Frontends,
						frontend)
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
			// do not continue if we already have a full connection active
			GLOBAL.backendData.RLock()
			for neighbor := range GLOBAL.backendData.SystemHashes[0] {
				if neighbor == remoteIP {
					backoutOnce.Do(killFunc)
					return
				}
			}
			// initialize nested systemHashes map if necessary
			_, ok := GLOBAL.backendData.SystemHashes[0][remoteIP]
			GLOBAL.backendData.RUnlock()
			if !ok {
				GLOBAL.backendData.Lock()
				for i := 0; i < 16; i++ {
					GLOBAL.backendData.SystemHashes[i][remoteIP] = map[string][4]byte{}
				}
				GLOBAL.backendData.Unlock()
			}

			dataReadySignal := make(chan struct{})
			rawData := []byte{}

			go func() {
				// wait until galaxyData is done mutating
				<-GLOBAL.mutatingDataBlocker

				// put together data
				outMsg := backendToBackendMessage{}
				outMsg.Priority = GLOBAL.serverPriority
				GLOBAL.bufferedRequestsMutex.RLock()
				outMsg.BufferedRequests = append(outMsg.BufferedRequests,
					GLOBAL.bufferedRequests...)
				outMsg.ConfirmedRequests = append(outMsg.ConfirmedRequests,
					GLOBAL.backendData.ConfirmedRequestIDs[localIP]...)
				GLOBAL.bufferedRequestsMutex.RUnlock()

				systemNames := []string{}
				maxBytes := 125000000 / 5 // (gigabit / 5) is 200mbit
				currentBytes := 0
				for i := range GLOBAL.backendData.SystemHashes {
					for sysName, hash := range GLOBAL.backendData.SystemHashes[i]["0.0.0.0"] {
						outMsg.SystemHashes[i][sysName] = hash
					}
				}
				for i := range GLOBAL.backendData.SystemHashes {
					for sysName, remoteHash := range GLOBAL.backendData.SystemHashes[i][localIP] {
						localHash := GLOBAL.backendData.SystemHashes[0]["0.0.0.0"][sysName]
						if reflect.DeepEqual(remoteHash, localHash) == false {
							systemNames = append(systemNames, sysName)
							currentBytes += len(GLOBAL.backendData.SystemBytes[sysName])
						}
						if currentBytes > maxBytes {
							break
						}
					}
				}
				rnum := getCryptoRnum()
				randomizedIndexes := rnum.Perm(len(systemNames) - 1)
				for _, rndIdx := range randomizedIndexes {
					sysName := systemNames[rndIdx]
					outMsg.SystemBytes[sysName] = append(outMsg.SystemBytes[sysName],
						GLOBAL.backendData.SystemBytes[sysName]...)
				}
				GLOBAL.backendData.RUnlock()

				// begin serializing the data
				networkOut.Reset()
				err := enc.Encode(outMsg)
				if checkError(err, fName, "error encoding outgoing message") {
					backoutOnce.Do(killFunc)
					return
				}

				// convert buffer to raw bytes
				rawData = networkOut.Bytes()

				// signal to the select branch, unblocking execution
				close(dataReadySignal)
			}()

			select {
			case <-GLOBAL.turningDataSignal:
				logError(fName, "ran out of time to send data to backendserver")
				<-dataReadySignal // don't want multiple goroutines mutating data
				continue
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
