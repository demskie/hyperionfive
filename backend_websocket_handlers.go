package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

func handleClientToBackend(w http.ResponseWriter, r *http.Request) {

	fName := "(clientToBackend) "

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(fName, client, "challenge") != true {
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
	if checkError(fName+"upgrade error:", err) {
		return
	}
	logDebug(fName+"opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[client]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[client]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[client] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[client]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive and process google authentication
	////////////////////////////////////////////////////////////

	userid := handleGoogleAuth(fName, conn, client)
	if userid == "" {
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
	// send data to client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			data := []byte{}
			dataReadySignal := make(chan struct{})
			go func() {
				// wait for new tick
				<-GLOBAL.newTickSignal
				// put together data
				playerData := Galaxy{}
				playerData.Name = GLOBAL.currentGalaxy.GetName()
				playerData.GlobalX = GLOBAL.currentGalaxy.GetGlobalX()
				playerData.GlobalY = GLOBAL.currentGalaxy.GetGlobalY()
				playerData.Type = GLOBAL.currentGalaxy.GetType()
				playerData.Solarsystemslice = GLOBAL.parsedPlayerPtrs[userid]

				data, err = proto.Marshal(&playerData)
				if checkError("CRITICAL FAILURE - UNABLE TO MARSHAL PLAYERDATA: ", err) {
					backoutOnce.Do(killFunc)
					return
				}
				// signal that data is ready
				close(dataReadySignal)
			}()

			select {
			case <-dataReadySignal:
				err = conn.WriteMessage(websocket.BinaryMessage, data)
				if checkError(fName+"error sending message to client:", err) {
					backoutOnce.Do(killFunc)
					return
				}
			case <-GLOBAL.turningDataSignal:
				logError("CRITICAL FAILURE - RAN OUT OF TIME TO SEND PLAYERDATA")
				backoutOnce.Do(killFunc)
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
			case <-backoutSignal:
				return
			default:
				inMsg := incomingClientRequest{}
				err = conn.ReadJSON(&inMsg)

				inMsg.ClientID = userid
				if checkError(fName+"error receiving request from Client:", err) {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName+"received the following JSON client request:\n",
					fmt.Sprintf("%v\n", inMsg))

				if inMsg.UniqueID == 0 {
					logError(fName + "Client tried to set the UniqueID")
					backoutOnce.Do(killFunc)
					return
				}
				inMsg.UniqueID = <-GLOBAL.uniqueRequestID

				GLOBAL.bufferedRequestsMutex.Lock()
				GLOBAL.bufferedRequests = append(GLOBAL.bufferedRequests, inMsg)
				GLOBAL.bufferedRequestsMutex.Unlock()
			}
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

	fName := "(backendToBackend) "

	// if client is blocked - backout of this http connection
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	thisCheckPassed := checkFirewall(fName, client, "challenge")
	if thisCheckPassed != true {
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(fName+"upgrade error:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// defer statements
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[client]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[client]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[client] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[client]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	logDebug(fName+"opening socket:", string(r.RemoteAddr))
	defer logDebug(fName+"closing connection:", client)

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse(fName, conn, client)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	type message struct {
		Priority          uint64
		currentGalaxyHash []byte
		ChangedGalaxyData *Galaxy
	}

	var (
		inMsg = message{}
		//outMsg = message{}
	)

	////////////////////////////////////////////////////////////
	// wait for message from Backend
	////////////////////////////////////////////////////////////

	msgType, newData, err := conn.ReadMessage()
	if checkError(fName+"error receiving message:", err) {
		return
	}
	if msgType != websocket.BinaryMessage {
		logInfo(fName + "messageType is invalid")
		return
	}
	network := bytes.NewBuffer(newData)
	dec := gob.NewDecoder(network)
	err = dec.Decode(&inMsg)
	if checkError(fName+"error decoding incoming message", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// process incoming message
	////////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////////
	// put together outgoing message
	////////////////////////////////////////////////////////////

}
