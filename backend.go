package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startBackendMode() {

	// initializing main data structures, locks, etc
	currentGalaxy = Galaxy{}

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

	challengeResponseWithLoadbalancer := func() bool {

		////////////////////////////////////////////////////////////
		// open websocket with loadbalancer and initiate two-way challenge-response authentication
		////////////////////////////////////////////////////////////

		// create websocket with loadbalancer frontend
		var dialer *websocket.Dialer
		conn, _, err := dialer.Dial("wss://hyperion5.com/backend", nil)
		defer conn.Close()
		if checkError("(backend_init) unable to open websocket with frontend loadbalancer:", err) {
			return false
		}

		// generate 256 random bytes
		randomBytes := make([]byte, 256)
		_, err = cryptorand.Read(randomBytes)
		if checkError("(backend_init) error while generating bytes:", err) {
			return false
		}
		logDebug("(backend_init) generated the following challenge:\n", fmt.Sprintf("%s", randomBytes))

		// if debug: convert randomBytes to randomString and print
		if debugModeBool == true {
			randomString := base64.StdEncoding.EncodeToString(randomBytes)
			logDebug("(backend_init) sent the following challenge:", randomString)
		}

		// send random string to frontend
		err = conn.WriteMessage(websocket.TextMessage, []byte(randomBytes))
		if checkError("(backend_init) error sending message to frontend loadbalancer:", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// parse incoming response
		////////////////////////////////////////////////////////////

		type incomingMessage struct {
			Response  string `json:"response"`
			Challenge string `json:"challenge"`
		}

		var inMsg incomingMessage
		err = conn.ReadJSON(&inMsg)
		if checkError("(backend_init) error receiving initial response from frontend loadbalancer:", err) == true {
			return false
		}
		if len(inMsg.Challenge) != 256 {
			logInfo("(backendsocket) received a challenge with invalid length:", string(len(inMsg.Challenge)), "bytes")
			return false
		}
		logDebug("(backend_init) received the following challenge:\n", fmt.Sprintf("%s", inMsg))

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
			logDebug("(backend_init) local hash is:\n", randomString)
		}

		// compare locally generated hash to received hash
		remoteHASH := []byte(inMsg.Response)
		if hmac.Equal(expectedHASH, remoteHASH) != true {
			logInfo("(backend_init) initial challenge-response failed:")
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
			logDebug("(backend_init) local hash for challenge sent by LoadBalancer:\n", randomString)
		}

		// send response to frontend
		err = conn.WriteMessage(websocket.TextMessage, []byte(response))
		if checkError("(backend_init) error sending message to frontend loadbalancer:", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// wait for response saying okay
		////////////////////////////////////////////////////////////
		_, data, err := conn.ReadMessage()
		if checkError("(backend_init) error receiving message to frontend loadbalancer:", err) {
			return false
		}

		message := base64.StdEncoding.EncodeToString(data)
		if message == "hellothere" {
			return true
		}

		return false
	}

	if challengeResponseWithLoadbalancer() != true {
		return
	}
	logDebug("(backend_init) successfully completed handshake with loadbalancer")

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

	// handle output file parameters
	// is the outputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
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
