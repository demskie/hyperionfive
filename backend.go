package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startBackendMode() {

	// easy on/off handle for debugging purposes
	var enabled bool

	////////////////////////////////////////////////////////////
	// process CLI parameters to potentially import data
	////////////////////////////////////////////////////////////

	enabled = false
	if enabled {
		handleInputParameters("startBackendMode")
	}

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	enabled = true
	if enabled {
		go func() {
			for {
				kickoffFrontendCommunication("hyperion5.com")
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// generate Galaxy
	////////////////////////////////////////////////////////////

	enabled = false
	if enabled {
		currentGalaxy = Galaxy{}
		futureGalaxy = Galaxy{}

		galaxyType := ELLIPTICAL
		generateGalaxy(galaxyType)

		logInfo(fmt.Sprint("   ...\tDeleting Rtree"))
		rtree = &rtreego.Rtree{}
		rtreeObjects = []rtreego.Spatial{}
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffFrontendCommunication(host string) {

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError("(handshakeFrontend) unable to open websocket"+
		"with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	// log client IP address
	logDebug("(handshakeFrontend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// initiate two-way challenge-response authentication
	////////////////////////////////////////////////////////////

	// define function to generate 256 bits of randomness
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if checkError("(handshakeFrontend) error while generating key", err) {
			conn.Close()
		}
		return key
	}

	// call above function and store value
	randomBytes := newHMACKey()
	logDebug("(handshakeFrontend) sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	// send random string to frontend
	err = conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if checkError("(handshakeFrontend) error sending"+
		" message to frontend loadbalancer:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// parse incoming response
	////////////////////////////////////////////////////////////

	type incomingChallengeResponse struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var inMsg incomingChallengeResponse
	err = conn.ReadJSON(&inMsg)
	if checkError("(handshakeFrontend) error receiving initial"+
		"response from frontend loadbalancer:", err) {
		return
	}

	logDebug("(handshakeFrontend) received the following JSON challenge/response:\n",
		fmt.Sprintf("%s\n", inMsg))

	////////////////////////////////////////////////////////////
	// validate incoming response to our challenge
	////////////////////////////////////////////////////////////

	// define function to produce symmetric signature using a shared secret key
	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	// convert data to array
	var dataChallengeArray [32]byte
	copy(dataChallengeArray[:], randomBytes[:])

	// generate response hash based on challenge and our secret password
	var correctResponse []byte
	correctResponse = generateHMAC([]byte(authPassword), dataChallengeArray)

	// compare locally generated hash to received hash
	remoteHash, _ := base64.StdEncoding.DecodeString(inMsg.Response)
	if hmac.Equal(correctResponse, remoteHash) != true {
		logInfo("(handshakeFrontend) initial challenge-response failed:",
			"\n", base64.StdEncoding.EncodeToString(correctResponse),
			"\n", base64.StdEncoding.EncodeToString(remoteHash))
		return
	}

	////////////////////////////////////////////////////////////
	// respond to challenge sent to us
	////////////////////////////////////////////////////////////

	// decode into binary
	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	// convert data to array
	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	// generate response hash based on challenge and our secret password
	ourResponse := generateHMAC([]byte(authPassword), remoteChallengeArray)
	logDebug("(handshakeFrontend) local hash for challenge sent by LoadBalancer:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	// send response to frontend
	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if checkError("(handshakeFrontend) error sending message"+
		"to frontend loadbalancer:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// initialize messages we'll be sending/receiving with frontend
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

		// put together the pending outgoingMessage
		netAddr := conn.LocalAddr()
		ipAddress, _, _ := net.SplitHostPort(netAddr.String())
		outMsgTemp.ServerAddress = ipAddress

		clientSessionCountMutex.RLock()
		outMsgTemp.ActiveClientCount = len(clientSessionCount)
		clientSessionCountMutex.RUnlock()

		// defining constants to satisfy my linter
		const (
			newgalaxy      = "new_galaxy"
			diediedie      = "diediedie"
			sameaslasttime = "same_as_last_time"
			yourgalaxyname = "your_galaxy_name"
		)

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = incomingMessageFrontend{}
		outMsgTemp = outgoingMessageFrontend{}

		////////////////////////////////////////////////////////////
		// seeing if we've been assigned a galaxy name (and acting upon it)
		////////////////////////////////////////////////////////////

		galName := currentGalaxy.Name
		logDebug(fmt.Sprint("(periodicUpdateFrontend) galName=\t\t", galName))

		if galName == "" {
			outMsgTemp.Message = newgalaxy
		}

		if galName != "" {
			galaxyInfo[galName].RLock()
			outMsgTemp.GalaxyName = galName
			outMsgTemp.TotalClientCount = len(galaxyInfo[galName].ClientIDs)
			outMsgTemp.OtherFrontends = galaxyInfo[galName].Frontends
			outMsgTemp.OtherBackends = galaxyInfo[galName].Backends
			galaxyInfo[galName].RUnlock()
		}

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) && galName != "" {
			outMsgTemp = outgoingMessageFrontend{Message: sameaslasttime}
		}

		err := conn.WriteJSON(outMsgTemp)
		if checkError("(periodicUpdateFrontend) error sending message:", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// receive incomingMessage
		////////////////////////////////////////////////////////////

		err = conn.ReadJSON(&inMsgTemp)
		if checkError("(periodicUpdateFrontend) error receiving message:", err) {
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
			galaxyInfo[galName] = &GalaxyInfoStruct{}
			logInfo(fmt.Sprintf("(periodicUpdateFrontend) received 'your_galaxy_name' = %s", galName))
		}

		if inMsgTemp.Message == diediedie {
			logInfo("(periodicUpdateFrontend) received 'diediedie'")
			gracefullyCloseBackend()
		}

		////////////////////////////////////////////////////////////
		// store incomingMessage and outgoingMessage for future reference
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

func backendWebSocket(w http.ResponseWriter, r *http.Request) {
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func gracefullyCloseBackend() {

	////////////////////////////////////////////////////////////
	// handle output parameters
	////////////////////////////////////////////////////////////

	foundMatch, _ := regexp.MatchString(".*blob$", inputFilename)
	if inputFilename == "" || foundMatch {
		protoOut, err := proto.Marshal(&currentGalaxy)
		if err != nil {
			log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
		}
		if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	} else {
		jsonOut, err := json.MarshalIndent(currentGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(outputFilename, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}

	// wait for logging to complete
	logInfo("PROGRAM EXITING GRACEFULLY")
	waitForLogging.Wait()

	// halt all further execution
	os.Exit(1)

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
