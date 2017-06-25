package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func spawnBackendToFrontend(host string) {
	fName := " backendToFrontend"
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError(fName+"unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

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

	type incomingMessageFrontend struct {
		GalaxyName     string   `json:"GalaxyName"`
		GlobalX        int64    `json:"GlobalX"`
		GlobalY        int64    `json:"GlobalY"`
		OtherFrontends []string `json:"OtherFrontends"`
		OtherBackends  []string `json:"OtherBackends"`
		Message        string   `json:"Message"`
		AnotherMessage string   `json:"AnotherMessage"`
	}
	type outgoingMessageFrontend struct {
		ServerAddress     string   `json:"ServerAddress"`
		GalaxyName        string   `json:"GalaxyName"`
		ActiveClientCount int      `json:"ActiveClientCount"`
		TotalClientCount  int      `json:"TotalClientCount"`
		OtherFrontends    []string `json:"OtherFrontends"`
		OtherBackends     []string `json:"OtherBackends"`
		Message           string   `json:"Message"`
		AnotherMessage    string   `json:"AnotherMessage"`
	}

	var (
		inMsgTemp       = incomingMessageFrontend{}
		inMsgReference  = incomingMessageFrontend{}
		outMsgTemp      = outgoingMessageFrontend{}
		outMsgReference = outgoingMessageFrontend{}
	)

	////////////////////////////////////////////////////////////
	// define function that will be responsible for sending/receiving
	////////////////////////////////////////////////////////////

	periodicUpdate := func() bool {

		inMsgTemp = incomingMessageFrontend{}
		outMsgTemp = outgoingMessageFrontend{}

		const (
			newgalaxy      = "new_galaxy"
			diediedie      = "diediedie"
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
			GLOBAL.backendDataMutex.Lock()
			outMsgTemp.GalaxyName = galName
			outMsgTemp.TotalClientCount = len(GLOBAL.backendData.ClientIDs)
			outMsgTemp.OtherFrontends = GLOBAL.backendData.Frontends
			outMsgTemp.OtherBackends = GLOBAL.backendData.Backends
			GLOBAL.backendDataMutex.Unlock()
		}

		netAddr := conn.LocalAddr()
		ipAddress, _, _ := net.SplitHostPort(netAddr.String())
		outMsgTemp.ServerAddress = ipAddress

		GLOBAL.clientSessionCountMutex.RLock()
		outMsgTemp.ActiveClientCount = len(GLOBAL.clientSessionCount)
		GLOBAL.clientSessionCountMutex.RUnlock()

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint(fName+"outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint(fName+"outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) && galName != "" {
			outMsgTemp = outgoingMessageFrontend{Message: sameaslasttime}
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
			GLOBAL.backendData = backendDataStruct{}
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
	// create loop to both send/receive
	////////////////////////////////////////////////////////////

	type message struct {
		Priority          uint64
		CurrentGalaxyHash []byte
		Requests          []incomingRequest
		RogerWilko        []int64
	}

	////////////////////////////////////////////////////////////
	// send data to backend
	////////////////////////////////////////////////////////////

	go func() {
		for {

			// do not continue if we already have a full connection active
			GLOBAL.fullBackendStatsMutex.RLock()
			for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
				if GLOBAL.fullBackendStats[i].ServerIP == host {
					return
				}
			}
			GLOBAL.fullBackendStatsMutex.RUnlock()

			dataReadySignal := make(chan struct{})
			rawData := []byte{}
			go func() {
				// wait for new tick
				<-GLOBAL.newTickSignal
				// put together data
				outMsg := message{}
				outMsg.Priority = GLOBAL.serverPriority
				outMsg.CurrentGalaxyHash = GLOBAL.currentGalaxyHash
				GLOBAL.bufferedRequestsMutex.RLock()
				outMsg.Requests = GLOBAL.bufferedRequests
				GLOBAL.bufferedRequestsMutex.RUnlock()
				GLOBAL.confirmedRequestsMutex.Lock()
				outMsg.RogerWilko = GLOBAL.confirmedRequests[host]
				GLOBAL.confirmedRequestsMutex.Unlock()

				// create buffer and begin serializing the data
				var networkOut bytes.Buffer
				networkOut.Reset()
				enc := gob.NewEncoder(&networkOut)
				err = enc.Encode(outMsg)
				if checkError(fName+"error encoding outgoing message", err) {
					backoutOnce.Do(killFunc)
					return
				}

				// convert buffer to raw bytes
				rawData = networkOut.Bytes()

				// When the data is received from the server
				// we are going to push

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
				return
			case <-backoutSignal:
				return
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// receive requests from client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			default:
				inMsg := message{}
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
				err = dec.Decode(&inMsg)
				if checkError(fName+"error decoding incoming message", err) {
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
