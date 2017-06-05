package main

import (
	"bufio"
	"bytes"
	"container/ring"
	"encoding/binary"
	"encoding/gob"
	"hash/crc32"
	"math"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

func handleClientToBackend(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall("cleintToBackend", client, "challenge") != true {
		return
	}

	////////////////////////////////////////////////////////////
	// upgrade websocket
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(clientToBackend) upgrade error:", err) {
		return
	}
	logDebug("(clientToBackend) opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientSessionCountMutex.Lock()
	_, ok := GLOBAL.clientSessionCount[client]
	if ok {
		GLOBAL.clientSessionCount[client]++
	} else {
		GLOBAL.clientSessionCount[client] = 1
	}
	GLOBAL.clientSessionCountMutex.Unlock()

	defer func() {
		GLOBAL.clientSessionCountMutex.Lock()
		GLOBAL.clientSessionCount[client]--
		GLOBAL.clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive and process google authentication
	////////////////////////////////////////////////////////////

	userid := handleGoogleAuth("clientToBackend", conn, client)
	if userid == "" {
		return
	}

	////////////////////////////////////////////////////////////
	// create channel to backout of nested gofuncs and for loops
	////////////////////////////////////////////////////////////

	backoutSignal := make(chan struct{})

	////////////////////////////////////////////////////////////
	// send data to client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			<-GLOBAL.processingSignal

			data := []byte{}
			dataReadySignal := make(chan struct{})
			go func() {
				playerData := Galaxy{}
				playerData.Name = GLOBAL.currentGalaxy.GetName()
				playerData.GlobalX = GLOBAL.currentGalaxy.GetGlobalX()
				playerData.GlobalY = GLOBAL.currentGalaxy.GetGlobalY()
				playerData.Type = GLOBAL.currentGalaxy.GetType()
				playerData.Solarsystemslice = GLOBAL.parsedPlayerPtrs[userid]
				data, err = proto.Marshal(&playerData)
				if checkError("CRITICAL FAILURE - UNABLE TO MARSHAL PLAYERDATA: ", err) {
					close(backoutSignal)
				} else {
					close(dataReadySignal)
				}
			}()
			select {
			case <-dataReadySignal:
				err := conn.WriteMessage(websocket.BinaryMessage, data)
				if checkError("(clientToBackend) error sending message to client:", err) {
					close(backoutSignal)
				}
			case <-GLOBAL.turningDataSignal:
				logError("CRITICAL FAILURE - RAN OUT OF TIME TO SEND PLAYERDATA")
				close(backoutSignal)
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
			newRequest := incomingRequest{ClientID: userid}

			GLOBAL.bufferedRequestsMutex.Lock()
			GLOBAL.bufferedRequests = append(GLOBAL.bufferedRequests, newRequest)
			GLOBAL.bufferedRequestsMutex.Unlock()

			time.Sleep(time.Second / 25)
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleBackendToBackend(w http.ResponseWriter, r *http.Request) {

	// if client is blocked - backout of this http connection
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	thisCheckPassed := checkFirewall("handshakeBackend", client, "challenge")
	if thisCheckPassed != true {
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(handshakeBackend) upgrade error:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// defer statements
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientSessionCountMutex.Lock()
	_, ok := GLOBAL.clientSessionCount[client]
	if ok {
		GLOBAL.clientSessionCount[client]++
	} else {
		GLOBAL.clientSessionCount[client] = 1
		logDebug("(handshakeBackend) setting 'clientSessionCount' to ONE",
			"for this client:", client)
	}
	GLOBAL.clientSessionCountMutex.Unlock()

	defer func() {
		GLOBAL.clientSessionCountMutex.Lock()
		GLOBAL.clientSessionCount[client]--
		GLOBAL.clientSessionCountMutex.Unlock()
	}()

	logDebug("(handshakeBackend) opening socket:", string(r.RemoteAddr))
	defer logDebug("(handshakeBackend) closing connection:", client)

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse("handshakeBackend", conn, client)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	type message struct {
		Source            uint32
		Priority          uint32
		SequenceNumber    uint64
		Checksum          []byte
		ChangedGalaxyData *Galaxy
	}

	var (
		inMsg  = message{}
		outMsg = message{}
	)

	////////////////////////////////////////////////////////////
	// wait for message from Backend
	////////////////////////////////////////////////////////////

	msgType, newData, err := conn.ReadMessage()
	if checkError("(periodicUpdateBackendGalaxy) error receiving message:", err) {
		return
	}
	if msgType != websocket.BinaryMessage {
		logInfo("(periodicUpdateBackendGalaxy) messageType is invalid")
		return
	}
	network := bytes.NewBuffer(newData)
	dec := gob.NewDecoder(network)
	err = dec.Decode(&inMsg)
	if checkError("(periodicUpdateBackendGalaxy) error decoding incoming message", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// process incoming message
	////////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////////
	// put together outgoing message
	////////////////////////////////////////////////////////////

	outMsg.SequenceNumber = atomic.LoadUint64(&GLOBAL.sequenceNumber)

	var result []byte
	branch := 2
	switch branch {

	case 1:
		data, err := proto.Marshal(&GLOBAL.currentGalaxy)
		if checkError("CRITICAL FAILURE - MARSHALLING GALAXY FOR CRC TEST: ", err) {
			os.Exit(1)
		}
		hash := crc32.NewIEEE()
		hash.Write(data)
		result = hash.Sum(nil)

	case 2:
		data, err := proto.Marshal(&GLOBAL.currentGalaxy)
		if checkError("CRITICAL FAILURE - MARSHALLING GALAXY FOR CRC TEST: ", err) {
			os.Exit(1)
		}
		buffer := bytes.NewBuffer(data)
		reader := bufio.NewReader(buffer)
		scanner := bufio.NewScanner(reader)

		hash := crc32.NewIEEE()
		for scanner.Scan() {
			hash.Write(scanner.Bytes())
		}
		result = hash.Sum(nil)

	}

	outMsg.Checksum = result

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handlePingPongBackend(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	if checkFirewall("pingPongBackend", client, "challenge") != true {
		return
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(pingPongBackend) upgrade error:", err) {
		return
	}
	logDebug("(pingPongBackend) opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientSessionCountMutex.Lock()
	_, ok := GLOBAL.clientSessionCount[client]
	if ok {
		GLOBAL.clientSessionCount[client]++
	} else {
		GLOBAL.clientSessionCount[client] = 1
	}
	GLOBAL.clientSessionCountMutex.Unlock()

	defer func() {
		GLOBAL.clientSessionCountMutex.Lock()
		GLOBAL.clientSessionCount[client]--
		GLOBAL.clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse("pingPongBackend", conn, client)
	if successful == false {
		return
	}

	var startTime int64
	var endTime int64
	var advertisedPriority uint64

	startTime = math.MaxInt64 // intentionally invalid

	for {

		GLOBAL.fullBackendStatsMutex.RLock()
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			if GLOBAL.fullBackendStats[i].ServerIP == client {
				return
			}
		}
		GLOBAL.fullBackendStatsMutex.RUnlock()

		msgType, newData, err := conn.ReadMessage()
		if checkError("(pingPongBackend) error receiving message:", err) {
			return
		}
		if msgType != websocket.BinaryMessage {
			logInfo("(pingPongBackend) messageType is invalid")
			return
		}

		if startTime != math.MaxInt64 {
			endTime = time.Now().UnixNano()
			latency := (endTime - startTime) - (backendTickrate.Nanoseconds() / 2)
			advertisedPriority = binary.LittleEndian.Uint64(newData)
			GLOBAL.idleBackendStatsMutex.Lock()
			idleBackendIndex := -1 // intentionally invalid
			for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
				if GLOBAL.idleBackendStats[i].ServerIP == client {
					idleBackendIndex = i
					break
				}
			}
			if idleBackendIndex == -1 {
				data := backendInfoStruct{}
				data.ServerIP = client
				data.AdvertisedPriority = advertisedPriority
				data.Latency = ring.New(latencyRingLength)
				data.Latency.Value = latency
				data.Latency = data.Latency.Next()
				GLOBAL.idleBackendStats = append(GLOBAL.idleBackendStats, data)
			} else {
				GLOBAL.idleBackendStats[idleBackendIndex].Latency.Value = latency
				GLOBAL.idleBackendStats[idleBackendIndex].Latency.Next()
				if GLOBAL.idleBackendStats[idleBackendIndex].AdvertisedPriority != advertisedPriority {
					logInfo("(pingPongBackend) advertisedPriority has changed (and isn't supposed to)")
				}
			}
			GLOBAL.idleBackendStatsMutex.Unlock()
		}

		GLOBAL.idleBackendStatsMutex.RLock()
		count := 0
		otherSideHasPriority := advertisedPriority > serverPriority
		for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
			if GLOBAL.idleBackendStats[i].ServerIP == client {
				count++
			}
			if count > 1 && otherSideHasPriority {
				return
			}
		}
		GLOBAL.idleBackendStatsMutex.RUnlock()

		time.Sleep(backendTickrate / 2)

		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, serverPriority)
		startTime = time.Now().UnixNano()
		err = conn.WriteMessage(websocket.BinaryMessage, data)
		if checkError("(pingPongBackend) error sending message to backend socket:", err) {
			return
		}
	}
}
