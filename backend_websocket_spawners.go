package main

import (
	"bytes"
	"container/ring"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

func spawnBackendToFrontend(host string) {

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError("(handshakeFrontend) unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	logDebug("(handshakeFrontend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := kickoffChallengeResponse("handshakeFrontend", conn)
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
		logDebug(fmt.Sprint("(periodicUpdateFrontend) galName=\t\t", galName))

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

		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) && galName != "" {
			outMsgTemp = outgoingMessageFrontend{Message: sameaslasttime}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err := enc.Encode(outMsgTemp)
		if checkError("(periodicUpdateFrontend) error encoding outgoing message", err) {
			return false
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// receive incomingMessage
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError("(periodicUpdateFrontend) error receiving message:", err) {
			return false
		}
		if msgType != websocket.BinaryMessage {
			logInfo("(periodicUpdateFrontend) messageType is invalid")
			return false
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError("(periodicUpdateFrontend) error decoding incoming message", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// process incomingMessage
		////////////////////////////////////////////////////////////

		if reflect.DeepEqual(inMsgTemp, inMsgReference) && inMsgTemp.Message != sameaslasttime {
			logInfo("(periodicUpdateFrontend) received duplicate message")
		}

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == yourgalaxyname && galName == "" {
			galName = inMsgTemp.GalaxyName
			GLOBAL.currentGalaxy.Name = galName
			GLOBAL.currentGalaxy.GlobalX = inMsgTemp.GlobalX
			GLOBAL.currentGalaxy.GlobalY = inMsgTemp.GlobalY
			GLOBAL.backendData = backendDataStruct{}
			logInfo(fmt.Sprintf("(periodicUpdateFrontend) received 'your_galaxy_name' = %s", galName))
		}

		if inMsgTemp.Message == yourgalaxyname && galName != "" {
			logInfo("(periodicUpdateFrontend) received 'your_galaxy_name' after already being assigned one")
		}

		if inMsgTemp.Message == diediedie {
			logInfo("(periodicUpdateFrontend) received 'diediedie'")
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

	if kickoffChallengeResponse(fName, conn) != true {
		return
	}

	for {

		GLOBAL.fullBackendStatsMutex.RLock()
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			if GLOBAL.fullBackendStats[i].ServerIP == host {
				return
			}
		}
		GLOBAL.fullBackendStatsMutex.RUnlock()

		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, serverPriority)
		startTime := time.Now().UnixNano()
		err = conn.WriteMessage(websocket.BinaryMessage, data)
		if checkError(fName+"error sending message to _data_ backend socket:", err) {
			return
		}

		msgType, newData, err := conn.ReadMessage()
		if checkError(fName+"error receiving message:", err) {
			return
		}
		endTime := time.Now().UnixNano()
		if msgType != websocket.BinaryMessage {
			logInfo(fName + "messageType is invalid")
			return
		}

		latency := (endTime - startTime) - (backendTickrate.Nanoseconds() / 2)
		advertisedPriority := binary.LittleEndian.Uint64(newData)

		GLOBAL.fullBackendStatsMutex.Lock()
		backendIndex := -1 // intentionally invalid
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			if GLOBAL.fullBackendStats[i].ServerIP == host {
				backendIndex = i
				break
			}
		}
		if backendIndex == -1 {
			data := backendInfoStruct{}
			data.ServerIP = host
			data.AdvertisedPriority = advertisedPriority
			data.Latency = ring.New(latencyRingLength)
			data.Latency.Value = latency
			data.Latency = data.Latency.Next()
			GLOBAL.fullBackendStats = append(GLOBAL.fullBackendStats, data)
		} else {
			GLOBAL.fullBackendStats[backendIndex].Latency.Value = latency
			GLOBAL.fullBackendStats[backendIndex].Latency.Next()
			if GLOBAL.fullBackendStats[backendIndex].AdvertisedPriority != advertisedPriority {
				logInfo(fName + "advertisedPriority has changed (and isn't supposed to)")
			}
		}
		GLOBAL.fullBackendStatsMutex.Unlock()

		GLOBAL.fullBackendStatsMutex.RLock()
		count := 0
		otherSideHasPriority := advertisedPriority > serverPriority
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			if GLOBAL.fullBackendStats[i].ServerIP == host {
				count++
			}
			if count > 1 && otherSideHasPriority {
				return
			}
		}
		GLOBAL.fullBackendStatsMutex.RUnlock()

		time.Sleep(backendTickrate / 2)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func spawnPingPongBackend(host string) {
	fName := "(pingPongBackend) "
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/pingpong", nil)
	if checkError(fName+"unable to open _idle_ websocket with backend:", err) {
		return
	}
	defer conn.Close()

	logDebug(fName+"opening connection:", conn.RemoteAddr().String())
	defer logDebug(fName+"closing connection:", conn.RemoteAddr().String())

	if kickoffChallengeResponse(fName, conn) != true {
		return
	}

	for {

		GLOBAL.fullBackendStatsMutex.RLock()
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			if GLOBAL.fullBackendStats[i].ServerIP == host {
				return
			}
		}
		GLOBAL.fullBackendStatsMutex.RUnlock()

		GLOBAL.idleBackendStatsMutex.RLock()
		for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
			if GLOBAL.idleBackendStats[i].ServerIP == host {
				return
			}
		}
		GLOBAL.idleBackendStatsMutex.RUnlock()

		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, serverPriority)
		startTime := time.Now().UnixNano()
		err = conn.WriteMessage(websocket.BinaryMessage, data)
		if checkError(fName+"error sending message to _idle_ backend socket:", err) {
			return
		}

		msgType, newData, err := conn.ReadMessage()
		if checkError(fName+"error receiving message:", err) {
			return
		}
		endTime := time.Now().UnixNano()
		if msgType != websocket.BinaryMessage {
			logInfo(fName + "messageType is invalid")
			return
		}

		latency := (endTime - startTime) - (backendTickrate.Nanoseconds() / 2)
		advertisedPriority := binary.LittleEndian.Uint64(newData)

		GLOBAL.idleBackendStatsMutex.Lock()
		idleBackendIndex := -1 // intentionally invalid
		for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
			if GLOBAL.idleBackendStats[i].ServerIP == host {
				idleBackendIndex = i
				break
			}
		}
		if idleBackendIndex == -1 {
			data := backendInfoStruct{}
			data.ServerIP = host
			data.AdvertisedPriority = advertisedPriority
			data.Latency = ring.New(latencyRingLength)
			data.Latency.Value = latency
			data.Latency = data.Latency.Next()
			GLOBAL.idleBackendStats = append(GLOBAL.idleBackendStats, data)
		} else {
			GLOBAL.idleBackendStats[idleBackendIndex].Latency.Value = latency
			GLOBAL.idleBackendStats[idleBackendIndex].Latency.Next()
			if GLOBAL.idleBackendStats[idleBackendIndex].AdvertisedPriority != advertisedPriority {
				logInfo(fName + "advertisedPriority has changed (and isn't supposed to)")
			}
		}
		GLOBAL.idleBackendStatsMutex.Unlock()

		GLOBAL.idleBackendStatsMutex.RLock()
		count := 0
		otherSideHasPriority := advertisedPriority > serverPriority
		for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
			if GLOBAL.idleBackendStats[i].ServerIP == host {
				count++
			}
			if count > 1 && otherSideHasPriority {
				return
			}
		}
		GLOBAL.idleBackendStatsMutex.RUnlock()

		time.Sleep(backendTickrate / 2)
	}
}
