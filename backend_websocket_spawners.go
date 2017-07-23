package main

import (
	"bytes"
	"container/ring"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"sync"
	"time"

	cryptorand "crypto/rand"

	"github.com/gorilla/websocket"
)

func spawnBackendToFrontend(host string) {

	fName := "(backendToFrontend) "

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError(fName+"unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	// remoteIPAddr, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	logDebug(fName+"opening connection:", conn.RemoteAddr().String())
	defer logDebug(fName+"closing connection:", conn.RemoteAddr().String())

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
			diediedie      = "diediedie" // TODO: ADD UNDERSCORES
			sameaslasttime = "same_as_last_time"
			yourgalaxyname = "your_galaxy_name"
		)

		////////////////////////////////////////////////////////////
		// seeing if we've been assigned a galaxy name (and acting upon it)
		////////////////////////////////////////////////////////////

		galName := GLOBAL.currentGalaxy.Name
		logDebug(fmt.Sprint(fName+"galName=\t\t", galName))

		if galName == "" {
			outMsgTemp.Message = newgalaxy
		}

		if galName != "" {
			GLOBAL.backendData.RLock()
			outMsgTemp.GalaxyName = galName
			//outMsgTemp.TotalClientCount = len(GLOBAL.backendData.ClientIDs)
			outMsgTemp.OtherFrontends = GLOBAL.backendData.Frontends
			otherBackends := []string{}
			for serverIP := range GLOBAL.backendData.RemoteSystemHashes {
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

		logDebug(fmt.Sprint(fName+"outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint(fName+"outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) && galName != "" {
			outMsgTemp = backendToFrontendMessage{
				Message: sameaslasttime,
			}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err := enc.Encode(outMsgTemp)
		if checkError(fName+"error encoding outgoing message", err) {
			return false
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// receive incomingMessage
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError(fName+"error receiving message:", err) {
			return false
		}
		if msgType != websocket.BinaryMessage {
			logInfo(fName + "messageType is invalid")
			return false
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError(fName+"error decoding incoming message", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// process incomingMessage
		////////////////////////////////////////////////////////////

		if reflect.DeepEqual(inMsgTemp, inMsgReference) && inMsgTemp.Message != sameaslasttime {
			logInfo(fName + "received duplicate message")
		}

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fmt.Sprint(fName+"inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint(fName+"inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == yourgalaxyname && galName == "" {
			galName = inMsgTemp.GalaxyName
			GLOBAL.currentGalaxy.Name = galName
			GLOBAL.currentGalaxy.GlobalX = inMsgTemp.GlobalX
			GLOBAL.currentGalaxy.GlobalY = inMsgTemp.GlobalY
			logInfo(fmt.Sprintf(fName+"received 'your_galaxy_name' = %s", galName))
		}

		if inMsgTemp.Message == yourgalaxyname && galName != "" {
			logInfo(fName + "received 'your_galaxy_name' after already being assigned one")
		}

		if inMsgTemp.Message == diediedie {
			logInfo(fName + "received 'diediedie'")
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

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backendsyncinfo", nil)
	if checkError(fName+"unable to open _data_ websocket with backend:", err) {
		return
	}
	defer conn.Close()

	localIPAddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	remoteIPAddr, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	logDebug(fName+"opening connection:", conn.RemoteAddr().String())
	defer logDebug(fName+"closing connection:", conn.RemoteAddr().String())

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
			for key := range GLOBAL.backendData.RemoteSystemHashes {
				if key == remoteIPAddr {
					return
				}
			}
			// initialize nested maps if necessary
			_, ok := GLOBAL.backendData.RemoteSystemHashes[remoteIPAddr]
			GLOBAL.backendData.RUnlock()
			if !ok {
				GLOBAL.backendData.Lock()
				if !ok {
					GLOBAL.backendData.RemoteSystemHashes[remoteIPAddr] = map[string][]byte{}
				}
				GLOBAL.backendData.Unlock()
			}

			dataReadySignal := make(chan struct{})
			networkOut := bytes.Buffer{}
			rawData := []byte{}
			go func() {
				// wait for new tick
				<-GLOBAL.newTickSignal

				// put together data
				outMsg := backendToBackendMessage{}

				outMsg.Priority = GLOBAL.serverPriority

				for sysName, hash := range GLOBAL.currentSystemHashes {
					outMsg.CurrentSystemHashes[sysName] = hash
				}

				GLOBAL.bufferedRequestsMutex.RLock()
				copy(outMsg.BufferedRequests, GLOBAL.bufferedRequests)
				GLOBAL.bufferedRequestsMutex.RUnlock()

				GLOBAL.confirmedRequestsMutex.Lock()
				copy(outMsg.ConfirmedRequests, GLOBAL.confirmedRequests[localIPAddr])
				GLOBAL.confirmedRequests[localIPAddr] = []int64{}
				GLOBAL.confirmedRequestsMutex.Unlock()

				localHashes := getCurrentSystemHashes()
				systemNames := []string{}
				maxBytes := 125000000 / 25 // .025gbit
				currentBytes := 0
				GLOBAL.backendData.RLock()
				for sysName, remoteHash := range GLOBAL.backendData.RemoteSystemHashes[remoteIPAddr] {
					localHash := localHashes[sysName]
					if reflect.DeepEqual(remoteHash, localHash) == false {
						systemNames = append(systemNames, sysName)
						currentBytes += len(localHashes[sysName])
					}
					if currentBytes > maxBytes {
						break
					}
				}
				seed := func() rand.Source {
					var randomness int64
					b := make([]byte, 8)
					cryptorand.Read(b)
					buf := bytes.NewBuffer(b)
					binary.Read(buf, binary.LittleEndian, &randomness)
					return rand.NewSource(randomness)
				}
				rnum := rand.New(seed())
				randomizedIndexes := rnum.Perm(len(systemNames) - 1)
				for _, rndIdx := range randomizedIndexes {
					sysName := systemNames[rndIdx]
					outMsg.SystemData = append(outMsg.SystemData, GLOBAL.currentSystemBytes[sysName])
				}
				GLOBAL.backendData.RUnlock()

				// begin serializing the data
				enc := gob.NewEncoder(&networkOut)
				err = enc.Encode(outMsg)
				if checkError(fName+"error encoding outgoing message", err) {
					backoutOnce.Do(killFunc)
					return
				}

				// convert buffer to raw bytes
				rawData = networkOut.Bytes()

				// signal to the select branch, unblocking execution
				close(dataReadySignal)
			}()

			select {
			case <-dataReadySignal:
				conn.WriteMessage(websocket.BinaryMessage, rawData)
				if checkError(fName+"error sending message to client:", err) {
					backoutOnce.Do(killFunc)
					return
				}
			case <-GLOBAL.turningDataSignal:
				logError("WARNING - RAN OUT OF TIME TO SEND DATA TO BACKENDSERVER")
				<-dataReadySignal
				return
			case <-backoutSignal:
				return
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// receive requests from other Backend server
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				msgType, newData, err := conn.ReadMessage()
				if checkError(fName+"error receiving message:", err) {
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
				if checkError(fName+"error decoding incoming message", err) {
					backoutOnce.Do(killFunc)
					return
				}

				GLOBAL.backendData.Lock()
				GLOBAL.backendData.RemoteSystemHashes[remoteIPAddr] = inMsg.CurrentSystemHashes
				_, ok := GLOBAL.backendData.RemoteAdvertisedPriority[remoteIPAddr]
				if !ok {
					GLOBAL.backendData.RemoteAdvertisedPriority[remoteIPAddr] = inMsg.Priority
				}
				_, ok = GLOBAL.backendData.RemoteAdvertisedPriority[remoteIPAddr]
				if !ok {
					GLOBAL.backendData.RemoteLatency[remoteIPAddr] = &ring.Ring{}
				}
				GLOBAL.backendData.Frontends = append(GLOBAL.backendData.Frontends,
					inMsg.Frontends...)
				oldPriority := GLOBAL.backendData.RemoteAdvertisedPriority[remoteIPAddr]
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
