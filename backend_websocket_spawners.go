package main

import (
	"bytes"
	"container/ring"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func spawnBackendToFrontend(host string) {
	fName := "(backendToFrontend)"

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError(err, fName, "unable to open websocket with frontend loadbalancer") {
		return
	}
	defer conn.Close()

	// remoteIPAddr, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	logDebug(fName, "opening connection:", conn.RemoteAddr().String())
	defer logDebug(fName, "closing connection:", conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := kickoffChallengeResponse(fName, conn)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structure for messages we'll be sending/receiving
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = frontendToBackendMessage{}
		inMsgReference  = frontendToBackendMessage{}
		outMsgTemp      = backendToFrontendMessage{}
		outMsgReference = backendToFrontendMessage{}
	)

	////////////////////////////////////////////////////////////
	// define function that will be responsible for sending/receiving
	////////////////////////////////////////////////////////////

	periodicUpdate := func() bool {

		inMsgTemp = frontendToBackendMessage{}
		outMsgTemp = backendToFrontendMessage{}

		const (
			newgalaxy      = "new_galaxy"
			diediedie      = "die_die_die"
			sameaslasttime = "same_as_last_time"
			yourgalaxyname = "your_galaxy_name"
		)

		////////////////////////////////////////////////////////////
		// seeing if we've been assigned a galaxy name (and acting upon it)
		////////////////////////////////////////////////////////////

		galName := GLOBAL.currentGalaxy.Name
		logDebug(fmt.Sprint(fName, "galName=\t\t", galName))

		if galName == "" {
			outMsgTemp.Message = newgalaxy
		}

		if galName != "" {
			GLOBAL.backendData.RLock()
			outMsgTemp.GalaxyName = galName
			outMsgTemp.OtherFrontends = GLOBAL.backendData.Frontends
			otherBackends := []string{}
			for serverIP := range GLOBAL.backendData.SystemHashes[0] {
				otherBackends = append(otherBackends, serverIP)
			}
			outMsgTemp.OtherBackends = otherBackends
			GLOBAL.backendData.RUnlock()
		}

		netAddr := conn.LocalAddr()
		myIPAddr, _, _ := net.SplitHostPort(netAddr.String())
		outMsgTemp.ServerAddress = myIPAddr

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint(fName, "outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint(fName, "outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) && galName != "" {
			outMsgTemp = backendToFrontendMessage{
				Message: sameaslasttime,
			}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err := enc.Encode(outMsgTemp)
		if checkError(err, fName, "error encoding outgoing message") {
			return false
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// receive incomingMessage
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError(err, fName, "error receiving message") {
			return false
		}
		if msgType != websocket.BinaryMessage {
			logInfo(fName, "messageType is invalid")
			return false
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError(err, fName, "error decoding incoming message") {
			return false
		}

		////////////////////////////////////////////////////////////
		// process incomingMessage
		////////////////////////////////////////////////////////////

		if reflect.DeepEqual(inMsgTemp, inMsgReference) && inMsgTemp.Message != sameaslasttime {
			logInfo(fName, "received duplicate message")
		}

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fmt.Sprint(fName, "inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint(fName, "inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == yourgalaxyname && galName == "" {
			galName = inMsgTemp.GalaxyName
			GLOBAL.currentGalaxy.Name = galName
			GLOBAL.currentGalaxy.GlobalX = inMsgTemp.GlobalX
			GLOBAL.currentGalaxy.GlobalY = inMsgTemp.GlobalY
			logInfo(fmt.Sprintf(fName, "received 'your_galaxy_name' = %s", galName))
		}

		if inMsgTemp.Message == yourgalaxyname && galName != "" {
			logInfo(fName, "received 'your_galaxy_name' after already being assigned one")
		}

		if inMsgTemp.Message == diediedie {
			logInfo(fName, "received 'diediedie'")
			gracefullyCloseBackend()
		}

		////////////////////////////////////////////////////////////
		// store messsages for future reference
		////////////////////////////////////////////////////////////

		inMsgReference = inMsgTemp

		if outMsgTemp.Message != sameaslasttime {
			outMsgReference = outMsgTemp
		}

		return true
	}

	////////////////////////////////////////////////////////////
	// loop periodicUpdate and fallback to calling Func if failure occurs
	////////////////////////////////////////////////////////////

	for {
		if periodicUpdate() == true {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	// fallback to calling Func
	time.Sleep(1 * time.Second)
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func spawnBackendToBackend(host string) {

	fName := "(backendToBackend) "

	dialer := &websocket.Dialer{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    8192, // 4096 is default
		WriteBufferSize:   8192, // 4096 is default
		EnableCompression: true,
	}
	conn, _, err := dialer.Dial("wss://"+host+":443/backendsync", nil)
	if checkError(err, fName, "unable to open websocket with backend") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1) // 1 is default

	localIPAddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	remoteIPAddr, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	logDebug(fName, "opening connection:", remoteIPAddr)
	defer logDebug(fName, "closing connection:", localIPAddr)

	////////////////////////////////////////////////////////////
	// initiate challenge-response handshake
	////////////////////////////////////////////////////////////

	if kickoffChallengeResponse(fName, conn) != true {
		return
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
	// send data to backend
	////////////////////////////////////////////////////////////

	go func() {

		for {
			// do not continue if we already have a full connection active
			GLOBAL.backendData.RLock()
			for neighbor := range GLOBAL.backendData.SystemHashes[0] {
				if neighbor == remoteIPAddr {
					killFunc()
					return
				}
			}
			// initialize nested systemHashes map if necessary
			_, ok := GLOBAL.backendData.SystemHashes[0][remoteIPAddr]
			GLOBAL.backendData.RUnlock()
			if !ok {
				GLOBAL.backendData.Lock()
				for i := 0; i < 16; i++ {
					GLOBAL.backendData.SystemHashes[i][remoteIPAddr] = map[string][4]byte{}
				}
				GLOBAL.backendData.Unlock()
			}

			dataReadySignal := make(chan struct{})
			networkOut := bytes.Buffer{}
			rawData := []byte{}

			go func() {
				// wait until galaxyData is done mutating
				<-GLOBAL.mutatingDataBlocker

				// put together data
				outMsg := backendToBackendMessage{}
				outMsg.Priority = GLOBAL.serverPriority
				GLOBAL.bufferedRequestsMutex.RLock()
				copy(outMsg.BufferedRequests, GLOBAL.bufferedRequests)
				copy(outMsg.ConfirmedRequests, GLOBAL.backendData.ConfirmedRequestIDs[localIPAddr])
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
					for sysName, remoteHash := range GLOBAL.backendData.SystemHashes[i][remoteIPAddr] {
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
					outMsg.SystemBytes[sysName] = GLOBAL.backendData.SystemBytes[sysName]
				}
				GLOBAL.backendData.RUnlock()

				// begin serializing the data
				enc := gob.NewEncoder(&networkOut)
				err = enc.Encode(outMsg)
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
				logError(fName, "WARNING - RAN OUT OF TIME TO SEND DATA TO BACKENDSERVER")
				<-dataReadySignal // TODO: this may be unnecessary
				continue
			default:
			}
			select {
			case <-dataReadySignal:
				conn.WriteMessage(websocket.BinaryMessage, rawData)
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
	// receive galaxyData and requests from other Backend server
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
					logInfo(fName + "messageType is invalid")
					backoutOnce.Do(killFunc)
					return
				}
				networkIn := bytes.NewBuffer(newData)
				dec := gob.NewDecoder(networkIn)
				inMsg := backendToBackendMessage{}
				err = dec.Decode(&inMsg)
				if checkError(err, fName, "error decoding incoming message") {
					backoutOnce.Do(killFunc)
					return
				}

				GLOBAL.backendData.Lock()
				recordReceiptOfMessage(remoteIPAddr)
				for i := range GLOBAL.backendData.SystemHashes {
					for sysName, localHash := range inMsg.SystemHashes[i] {
						GLOBAL.backendData.SystemHashes[i][remoteIPAddr][sysName] = localHash
					}
				}
				_, ok := GLOBAL.backendData.Priorities[remoteIPAddr]
				if !ok {
					GLOBAL.backendData.Priorities[remoteIPAddr] = inMsg.Priority
				}
				_, ok = GLOBAL.backendData.Priorities[remoteIPAddr]
				if !ok {
					GLOBAL.backendData.Latencies[remoteIPAddr] = &ring.Ring{}
				}
				GLOBAL.backendData.Frontends = append(GLOBAL.backendData.Frontends,
					inMsg.Frontends...)
				oldPriority := GLOBAL.backendData.Priorities[remoteIPAddr]
				GLOBAL.backendData.Unlock()

				if oldPriority != inMsg.Priority {
					logError(fName + "ERROR: The Advertised Priority changed from: " + remoteIPAddr)
					backoutOnce.Do(killFunc)
					return
				}
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	// TODO: NEED TO CLEANUP DATA AND INFO STRUCTS

	return
}
