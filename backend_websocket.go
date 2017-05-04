package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

func backendReceiveClientGalaxyRequest(w http.ResponseWriter, r *http.Request) {
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func backendReceiveBackendInfoRequest(w http.ResponseWriter, r *http.Request) {
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func backendReceiveBackendGalaxyRequest(w http.ResponseWriter, r *http.Request) {
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffFrontendInfo(host string) {

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError("(handshakeFrontend) unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	// log client IP address
	logDebug("(handshakeFrontend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	clientIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	successful := handleChallengeResponse("handshakeFrontend", conn, clientIP)
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

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = incomingMessageFrontend{}
		outMsgTemp = outgoingMessageFrontend{}

		////////////////////////////////////////////////////////////
		// defining constants to satisfy my linter
		////////////////////////////////////////////////////////////

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
	// loop periodicUpdate and fallback to main if failure occurs
	////////////////////////////////////////////////////////////

	for {
		if periodicUpdate() == true {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	// fallback to main
	time.Sleep(1 * time.Second)
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffBackendInfoSync(host string) {

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backendsyncinfo", nil)
	if checkError("(handshakeFrontend) unable to open _info_ websocket with backend:", err) {
		return
	}
	defer conn.Close()
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffBackendGalaxySync(host string) {

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backendsyncgalaxy", nil)
	if checkError("(handshakeFrontend) unable to open _galaxy_ websocket with backend:", err) {
		return
	}
	defer conn.Close()
}
