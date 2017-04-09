package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// global variables
var (
	randomSeed    *rand.Rand
	currentGalaxy Galaxy
	futureGalaxy  Galaxy

	sliceOfLocalPlayers []string

	partitionedSolarSystems []sync.RWMutex
)

func startBackendMode() {

	// initializing main data structures, locks, etc
	currentGalaxy = Galaxy{}
	futureGalaxy = Galaxy{}

	sliceOfLocalPlayers = []string{}

	// handle input file parameters
	// is the inputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	var doneMatching bool
	if inputFilename == "" {
		doneMatching = true
		logDebug("(backend_init) no inputfilename specified - skipping data import process")
	} else {
		doneMatching = false
	}
	// skip forward if doneMatching is 'true'
	// otherwise parse filename to determine if it is a 'blob' file
	// and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*blob$", inputFilename)
		if foundMatch == true {
			doneMatching = true
			logInfo("(backend_init) parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFilename)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln("data.blob not found")
				} else {
					log.Fatalln("Error reading file:", inputFilename, err)
				}
			}
			if err := proto.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// skip forward if doneMatching is 'true'
	// otherwise parse filename to determine if it is a 'json' file
	// and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*json$", inputFilename)
		if foundMatch == true {
			doneMatching = true
			logInfo("(backend_init) parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFilename)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln(inputFilename, "not found")
				} else {
					log.Fatalln("Error reading file:", inputFilename, err)
				}
			}
			if err := json.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// if none of the above logic was successful - we have a fatal error on our hands
	if doneMatching == false {
		log.Fatalln("(backend_init) unknown inputfilename - skipping data import process")
	}

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	numberFailedHandshakes := 0
	for {
		// create websocket with loadbalancer frontend
		var dialer *websocket.Dialer
		conn, _, err := dialer.Dial("wss://hyperion5.com/backend", nil)
		defer conn.Close()
		defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())
		if checkError("(handshakeFrontend) unable to open websocket"+
			"with frontend loadbalancer:", err) {
			return
		}
		if handshakeWithFrontend(conn) != true {
			logDebug("(handshakeFrontend) initial handshake with loadbalancer failed")
			numberFailedHandshakes++
			if numberFailedHandshakes > 3 {
				logDebug("(handshakeFrontend) just failed three handshakes in a row!!!")
				waitForLogging.Wait()
				os.Exit(1)
			} else {
				numberFailedHandshakes = 0
			}
		}

		var (
			inMsg     = incomingMessageFrontend{}
			inMsgOld  = incomingMessageFrontend{}
			outMsg    = outgoingMessageFrontend{}
			outMsgOld = outgoingMessageFrontend{}
		)

		for {
			if periodicUpdate(conn, &inMsg, &inMsgOld, &outMsg, &outMsgOld) != true {
				logDebug("(periodicUpdateFrontend) failed to periodicly receive",
					"or send data with loadbalancer")
				break // fall back to handshake
			}
			time.Sleep(1 * time.Second)
			inMsgOld = inMsg   // used later to check if incomingMessage has changed
			outMsgOld = outMsg // used later to check if outgoingMessage has changed
		}
	}

}

func handshakeWithFrontend(conn *websocket.Conn) bool {

	////////////////////////////////////////////////////////////
	// initiate two-way challenge-response authentication
	////////////////////////////////////////////////////////////

	// generate 256 random bytes
	randomBytes := make([]byte, 256)
	_, err := cryptorand.Read(randomBytes)
	if checkError("(handshakeFrontend) error while generating bytes:", err) {
		return false
	}
	logDebug("(handshakeFrontend) generated the following challenge:\n",
		fmt.Sprintf("%s", randomBytes))

	// if debug: convert randomBytes to randomString and print
	if debugModeBool == true {
		randomString := base64.StdEncoding.EncodeToString(randomBytes)
		logDebug("(handshakeFrontend) sent the following challenge:", randomString)
	}

	// send random string to frontend
	err = conn.WriteMessage(websocket.TextMessage, []byte(randomBytes))
	if checkError("(handshakeFrontend) error sending message to frontend loadbalancer:", err) {
		return false
	}

	////////////////////////////////////////////////////////////
	// parse incoming response
	////////////////////////////////////////////////////////////

	type incomingChallengeResponse struct {
		Response  string `json:"response"`
		Challenge string `json:"challenge"`
	}

	var inMsg incomingChallengeResponse
	err = conn.ReadJSON(&inMsg)
	if checkError("(handshakeFrontend) error receiving initial"+
		"response from frontend loadbalancer:", err) {
		return false
	}
	if len(inMsg.Challenge) != 256 {
		logInfo("(handshakeFrontend) received a challenge with invalid length:",
			string(len(inMsg.Challenge)), "bytes")
		return false
	}
	logDebug("(handshakeFrontend) received the following challenge:\n",
		fmt.Sprintf("%s", inMsg))

	////////////////////////////////////////////////////////////
	// validate incoming response to our challenge
	////////////////////////////////////////////////////////////

	// create local hash to compare with Response
	hash := hmac.New(sha512.New512_256, []byte(authPassword))
	hash.Write(randomBytes)
	expectedHASH := hash.Sum(nil)

	// if debug: print local hash
	if debugModeBool == true {
		randomString := base64.StdEncoding.EncodeToString(randomBytes)
		logDebug("(handshakeFrontend) local hash is:\n", randomString)
	}

	// compare locally generated hash to received hash
	remoteHASH := []byte(inMsg.Response)
	if hmac.Equal(expectedHASH, remoteHASH) != true {
		logInfo("(handshakeFrontend) initial challenge-response failed:")
		return false
	}

	////////////////////////////////////////////////////////////
	// respond to challenge sent to us
	////////////////////////////////////////////////////////////

	// create local hash using Challenge instead of locally generated bytes
	randomBytes = []byte(inMsg.Challenge)
	hash = hmac.New(sha512.New512_256, []byte(authPassword))
	hash.Write(randomBytes)
	response := hash.Sum(nil)

	// if debug: print local hash
	if debugModeBool == true {
		randomString := base64.StdEncoding.EncodeToString(randomBytes)
		logDebug("(handshakeFrontend) local hash for challenge sent by LoadBalancer:\n",
			randomString)
	}

	// send response to frontend
	err = conn.WriteMessage(websocket.TextMessage, []byte(response))
	if checkError("(handshakeFrontend) error sending message"+
		"to frontend loadbalancer:", err) {
		return false
	}

	////////////////////////////////////////////////////////////
	// wait for response saying okay
	////////////////////////////////////////////////////////////
	_, data, err := conn.ReadMessage()
	if checkError("(handshakeFrontend) error receiving message"+
		"from frontend loadbalancer:", err) {
		return false
	}

	message := base64.StdEncoding.EncodeToString(data)
	if message == "hellothere" {
		return true
	}

	return false
}

type incomingMessageFrontend struct {
	ListOfGalaxies []struct {
		GalaxyName string  `json:"GalaxyName"`
		GlobalX    float64 `json:"GlobalX"`
		GlobalY    float64 `json:"GlobalY"`
		Diameter   int32   `json:"Diameter"`
	} `json:"ListOfGalaxies"`
	OtherFrontends []string `json:"OtherFrontends"`
	OtherBackends  []string `json:"OtherBackends"`
	Message        string   `json:"Message"`
}
type outgoingMessageFrontend struct {
	ServerAddress     string   `json:"ServerAddress"`
	GalaxyName        string   `json:"GalaxyName"`
	ActiveClientCount int      `json:"ActiveClientCount"`
	TotalClientCount  int      `json:"TotalClientCount"`
	OtherFrontends    []string `json:"OtherFrontends"`
	OtherBackends     []string `json:"OtherBackends"`
	Message           string   `json:"Message"`
}

func periodicUpdate(myConn *websocket.Conn,
	inMsg *incomingMessageFrontend, inMsgOld *incomingMessageFrontend,
	outMsg *outgoingMessageFrontend, outMsgOld *outgoingMessageFrontend) bool {

	////////////////////////////////////////////////////////////
	// put together the pending outgoingMessage
	////////////////////////////////////////////////////////////

	netAddr := myConn.LocalAddr()
	ipAddress, _, _ := net.SplitHostPort(netAddr.String())
	outMsg.ServerAddress = ipAddress

	outMsg.GalaxyName = currentGalaxy.GetName()

	clientSessionCountMutex.RLock()
	outMsg.ActiveClientCount = len(clientSessionCount)
	clientSessionCountMutex.RUnlock()

	outMsg.TotalClientCount = len(sliceOfLocalPlayers)

	outMsg.OtherFrontends = sliceOfOtherFrontends

	outMsg.OtherBackends = sliceOfOtherFrontends

	////////////////////////////////////////////////////////////
	// if outMsg is the same as last time - send only message
	////////////////////////////////////////////////////////////

	if reflect.DeepEqual(outMsg, outMsgOld) {
		logDebug("(periodicUpdateFrontend) outgoing info hasn't changed")
		outMsg = &outgoingMessageFrontend{Message: "same_as_last_time"}
	}

	////////////////////////////////////////////////////////////
	// send outgoingMessage
	////////////////////////////////////////////////////////////

	err := myConn.WriteJSON(outMsg)
	if checkError("(periodicUpdateFrontend) error sending message:", err) {
		return false
	}

	////////////////////////////////////////////////////////////
	// receive and process incomingMessage
	////////////////////////////////////////////////////////////

	err = myConn.ReadJSON(&inMsg)
	if checkError("(periodicUpdateFrontend) error receiving message:", err) {
		return false
	}

	switch inMsg.Message {
	case "diediedie":
		logInfo("(periodicUpdateFrontend) received 'diediedie' from frontend")
		gracefullyCloseBackend()
	case "same_as_last_time":
		logInfo("(periodicUpdateFrontend) received 'same_as_last_time' from frontend")
		gracefullyCloseBackend()
	default:
		logInfo("(periodicUpdateFrontend) received unknown message from frontend:",
			inMsg.Message)
	}

	if reflect.DeepEqual(inMsg, inMsgOld) {
		logInfo("(periodicUpdateFrontend) received duplciate message from frontend")
	}

	return true
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
	waitForLogging.Wait()

	// halt all further execution
	os.Exit(1)

}
