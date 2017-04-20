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

	// easy on/off handle for debugging purposes
	enabled := true

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

	generateGalaxy()

	////////////////////////////////////////////////////////////
	// random sanity checks
	////////////////////////////////////////////////////////////

	logDebug(fmt.Sprintf("There are %v solarSystems", len(currentGalaxy.GetSolarsystemslice())))
	logDebug(fmt.Sprintf("First Galaxy Name: %s", currentGalaxy.Solarsystemslice[0].GetName()))
	logDebug(fmt.Sprintf("Random Galaxy Name: %s", currentGalaxy.Solarsystemslice[9183].Name))

	getSystemIndex := func(myName string) (int, bool) {
		for i, elem := range currentGalaxy.Solarsystemslice {
			if elem.GetName() == myName {
				return i, true
			}
		}
		return -1, false
	}

	if debugModeBool {
		testSystem, ok := getSystemIndex("xyz9999")
		if ok {
			logDebug(fmt.Sprintf("Printing name of 9999th system: %s\n",
				currentGalaxy.Solarsystemslice[testSystem].GetName()))
		}

	}

	////////////////////////////////////////////////////////////
	// basic sanity checks
	////////////////////////////////////////////////////////////

	//time.Sleep(30 * time.Second)
	//waitForLogging.Wait()

	//tempWaitgroup.Wait()

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
	conn, _, err := dialer.Dial("wss://"+host+"/backend", nil)
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
		ListOfGalaxies []struct {
			GalaxyName string `json:"GalaxyName"`
			GlobalX    int64  `json:"GlobalX"`
			GlobalY    int64  `json:"GlobalY"`
			Diameter   int32  `json:"Diameter"`
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

		outMsgTemp.GalaxyName = currentGalaxy.GetName()

		clientSessionCountMutex.RLock()
		outMsgTemp.ActiveClientCount = len(clientSessionCount)
		clientSessionCountMutex.RUnlock()

		outMsgTemp.TotalClientCount = len(sliceOfLocalPlayers)

		outMsgTemp.OtherFrontends = sliceOfOtherFrontends

		outMsgTemp.OtherBackends = sliceOfOtherFrontends

		////////////////////////////////////////////////////////////
		// if outMsgTemp is the same as last time - send only message
		////////////////////////////////////////////////////////////

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			//logDebug("(periodicUpdateFrontend) outgoing info hasn't changed")
			outMsgTemp = outgoingMessageFrontend{Message: "same_as_last_time"}
		}

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		err := conn.WriteJSON(outMsgTemp)
		if checkError("(periodicUpdateFrontend) error sending message:", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// receive and process incomingMessage
		////////////////////////////////////////////////////////////

		err = conn.ReadJSON(&inMsgTemp)
		if checkError("(periodicUpdateFrontend) error receiving message:", err) {
			return false
		}

		switch inMsgTemp.Message {
		case "diediedie":
			logInfo("(periodicUpdateFrontend) received 'diediedie'")
			gracefullyCloseBackend()
		case "same_as_last_time":
			//logDebug("(periodicUpdateFrontend) received 'same_as_last_time'")
		case "":
			// do nothing
		default:
			logInfo("(periodicUpdateFrontend) received unknown message:",
				inMsgTemp.Message)
		}

		// shouldn't receive duplicate message from Frontend - but we sanitycheck anyways
		if reflect.DeepEqual(inMsgTemp, inMsgReference) && inMsgTemp.Message != "same_as_last_time" {
			logInfo("(periodicUpdateFrontend) received duplicate message")
		}

		// used later to check if incomingMessage has changed
		inMsgReference = inMsgTemp
		outMsgReference = outMsgTemp

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
