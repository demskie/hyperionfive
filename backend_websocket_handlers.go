package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

func handleClientToBackend(w http.ResponseWriter, r *http.Request) {
	fName := "(clientToBackend)"

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	remoteIPAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(fName, remoteIPAddr, "challenge") != true {
		return
	}

	////////////////////////////////////////////////////////////
	// upgrade websocket
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}
	logDebug(fName, "opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	logDebug(fName, "opening socket:", remoteIPAddr)
	defer logDebug(fName, "closing connection:", remoteIPAddr)

	////////////////////////////////////////////////////////////
	// receive and process google authentication
	////////////////////////////////////////////////////////////

	userid := handleGoogleAuth(fName, conn, remoteIPAddr)
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
				GLOBAL.backendData.RLock()
				playerData.Solarsystemslice = GLOBAL.backendData.ParsedPlayerPtrs[userid]
				GLOBAL.backendData.RUnlock()
				playerData.Type = GLOBAL.currentGalaxy.GetType()

				// serialize this information
				data, err = proto.Marshal(&playerData)
				if checkError(err, fName, "CRITICAL FAILURE. UNABLE TO MARSHAL PLAYERDATA") {
					backoutOnce.Do(killFunc)
					return
				}

				// signal that data is ready
				close(dataReadySignal)
			}()

			select {
			case <-dataReadySignal:
				err = conn.WriteMessage(websocket.BinaryMessage, data)
				if checkError(err, fName, "error sending message to client") {
					backoutOnce.Do(killFunc)
					return
				}
			case <-GLOBAL.turningDataSignal:
				logError(fName, "CRITICAL FAILURE - RAN OUT OF TIME TO SEND PLAYERDATA")
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
				if checkError(err, fName, "error receiving request from Client") {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName, "received the following JSON client request:\n",
					fmt.Sprintf("%v\n", inMsg))

				if inMsg.UniqueID != 0 {
					logError(fName, "Client tried to set the UniqueID:", remoteIPAddr)
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
	fName := "(backendToBackend)"

	// if client is blocked - backout of this http connection
	remoteIPAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	thisCheckPassed := checkFirewall(fName, remoteIPAddr, "challenge")
	if thisCheckPassed != true {
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}

	////////////////////////////////////////////////////////////
	// defer statements
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	logDebug(fName+"opening socket:", remoteIPAddr)
	defer logDebug(fName+"closing connection:", remoteIPAddr)

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse(fName, conn, remoteIPAddr)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////////
	// wait for message from Backend
	////////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////////
	// process incoming message
	////////////////////////////////////////////////////////////

	////////////////////////////////////////////////////////////
	// put together outgoing message
	////////////////////////////////////////////////////////////

}
