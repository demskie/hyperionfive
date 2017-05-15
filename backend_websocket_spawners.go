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

		galName := currentGalaxy.Name
		logDebug(fmt.Sprint("(periodicUpdateFrontend) galName=\t\t", galName))

		if galName == "" {
			outMsgTemp.Message = newgalaxy
		}

		if galName != "" {
			backendDataMutex.Lock()
			outMsgTemp.GalaxyName = galName
			outMsgTemp.TotalClientCount = len(backendData.ClientIDs)
			outMsgTemp.OtherFrontends = backendData.Frontends
			outMsgTemp.OtherBackends = backendData.Backends
			backendDataMutex.Unlock()
		}

		netAddr := conn.LocalAddr()
		ipAddress, _, _ := net.SplitHostPort(netAddr.String())
		outMsgTemp.ServerAddress = ipAddress

		clientSessionCountMutex.RLock()
		outMsgTemp.ActiveClientCount = len(clientSessionCount)
		clientSessionCountMutex.RUnlock()

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
			currentGalaxy.Name = galName
			currentGalaxy.GlobalX = inMsgTemp.GlobalX
			currentGalaxy.GlobalY = inMsgTemp.GlobalY
			backendData = BackendDataStruct{}
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

func spawnBackendToBackend(host string) {

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backendsyncinfo", nil)
	if checkError("(handshakeBackend) unable to open _data_ websocket with backend:", err) {
		return
	}
	defer conn.Close()
}

func spawnPingPongBackend(host string) {

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/pingpong", nil)
	if checkError("(pingPongBackend) unable to open _idle_ websocket with backend:", err) {
		return
	}
	defer conn.Close()

	logDebug("(pingPongBackend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(pingPongBackend) closing connection:", conn.RemoteAddr().String())

	if kickoffChallengeResponse("pingPongBackend", conn) != true {
		return
	}

	for {

		connectedBackendsMutex.RLock()
		for i := 0; i < len(connectedBackends); i++ {
			if connectedBackends[i].ServerIP == host {
				return
			}
		}
		connectedBackendsMutex.RUnlock()

		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, serverPriority)
		startTime := time.Now().UnixNano()
		err = conn.WriteMessage(websocket.BinaryMessage, data)
		if checkError("(pingPongBackend) error sending message to _idle_ backend socket:", err) {
			return
		}

		msgType, newData, err := conn.ReadMessage()
		if checkError("(pingPongBackend) error receiving message:", err) {
			return
		}
		endTime := time.Now().UnixNano()
		if msgType != websocket.BinaryMessage {
			logInfo("(pingPongBackend) messageType is invalid")
			return
		}

		latency := (endTime - startTime) - (backendTickrate.Nanoseconds() / 2)
		advertisedPriority := binary.LittleEndian.Uint64(newData)

		idleConnectedBackendsMutex.Lock()
		idleBackendIndex := -1 // intentionally invalid
		for i := 0; i < len(idleConnectedBackends); i++ {
			if idleConnectedBackends[i].ServerIP == host {
				idleBackendIndex = i
				break
			}
		}
		if idleBackendIndex == -1 {
			data := Backend{}
			data.ServerIP = host
			data.AdvertisedPriority = advertisedPriority
			data.Latency = ring.New(latencyRingLength)
			data.Latency.Value = latency
			data.Latency = data.Latency.Next()
			idleConnectedBackends = append(idleConnectedBackends, data)
		} else {
			idleConnectedBackends[idleBackendIndex].Latency.Value = latency
			idleConnectedBackends[idleBackendIndex].Latency.Next()
			if idleConnectedBackends[idleBackendIndex].AdvertisedPriority != advertisedPriority {
				logInfo("(pingPongBackend) advertisedPriority has changed (and isn't supposed to)")
			}
		}
		idleConnectedBackendsMutex.Unlock()

		idleConnectedBackendsMutex.RLock()
		count := 0
		otherSideHasPriority := advertisedPriority > serverPriority
		for i := 0; i < len(idleConnectedBackends); i++ {
			if idleConnectedBackends[i].ServerIP == host {
				count++
			}
			if count > 1 && otherSideHasPriority {
				return
			}
		}
		idleConnectedBackendsMutex.RUnlock()

		time.Sleep(backendTickrate / 2)
	}
}
