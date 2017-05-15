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
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

func handleBackendToBackend(w http.ResponseWriter, r *http.Request) {

	// if client is blocked - backout of this http connection with 'return'
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

	clientSessionCountMutex.Lock()
	_, ok := clientSessionCount[client]
	if ok {
		clientSessionCount[client]++
	} else {
		clientSessionCount[client] = 1
		logDebug("(handshakeBackend) setting 'clientSessionCount' to ONE",
			"for this client:", client)
	}
	clientSessionCountMutex.Unlock()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
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

	outMsg.SequenceNumber = atomic.LoadUint64(&sequenceNumber)

	var result []byte
	branch := 2
	switch branch {

	case 1:
		data, err := proto.Marshal(&currentGalaxy)
		if checkError("CRITICAL FAILURE - MARSHALLING GALAXY FOR CRC TEST: ", err) {
			os.Exit(1)
		}
		hash := crc32.NewIEEE()
		hash.Write(data)
		result = hash.Sum(nil)

	case 2:
		data, err := proto.Marshal(&currentGalaxy)
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

	clientSessionCountMutex.Lock()
	_, ok := clientSessionCount[client]
	if ok {
		clientSessionCount[client]++
	} else {
		clientSessionCount[client] = 1
	}
	clientSessionCountMutex.Unlock()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse("pingPongBackend", conn, client)
	if successful == false {
		return
	}

	connectedBackendsMutex.RLock()
	for i := 0; i < len(connectedBackends); i++ {
		if connectedBackends[i].ServerIP == client {
			return
		}
	}
	connectedBackendsMutex.RUnlock()

	var startTime int64
	var endTime int64
	var advertisedPriority uint64

	startTime = math.MaxInt64

	for {

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
			connectedBackendsMutex.Lock()
			backendIndex := -1 // intentionally invalid
			for i := 0; i < len(connectedBackends); i++ {
				if connectedBackends[i].ServerIP == client {
					backendIndex = i
					break
				}
			}
			if backendIndex == -1 {
				data := Backend{}
				data.ServerIP = client
				data.AdvertisedPriority = advertisedPriority
				data.Latency = ring.New(latencyRingLength)
				data.Latency.Value = latency
				data.Latency = data.Latency.Next()
				connectedBackends = append(connectedBackends, data)
			} else {
				connectedBackends[backendIndex].Latency.Value = latency
				connectedBackends[backendIndex].Latency.Next()
				if connectedBackends[backendIndex].AdvertisedPriority != advertisedPriority {
					logInfo("(pingPongBackend) advertisedPriority has changed (and isn't supposed to)")
				}
			}
			connectedBackendsMutex.Unlock()
		}

		connectedBackendsMutex.RLock()
		count := 0
		otherSideHasPriority := advertisedPriority > serverPriority
		for i := 0; i < len(connectedBackends); i++ {
			if connectedBackends[i].ServerIP == client {
				count++
			}
			if count > 1 && otherSideHasPriority {
				return
			}
		}
		connectedBackendsMutex.RUnlock()

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
