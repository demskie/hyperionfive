package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"net"

	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

func handleChallengeResponse(fName string, conn *websocket.Conn) (result bool) {
	fName += " (challenge)"

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if checkFirewall(CHALLENGE, fName, remoteIP) != true {
		return false
	}

	////////////////////////////////////////////////////////////
	// receive initial challenge from remote end
	////////////////////////////////////////////////////////////

	_, dataChallenge, err := conn.ReadMessage()
	if checkError(err, fName, "error receiving challenge from backend") {
		return false
	}
	logDebug(fName, "received the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(dataChallenge))

	////////////////////////////////////////////////////////////
	// ensure that challenge is 256bits and cast to array if so
	////////////////////////////////////////////////////////////

	var dataChallengeArray [32]byte

	if len(dataChallenge) == 32 {
		copy(dataChallengeArray[:], dataChallenge)
	} else {
		logError(fName, "received a challenge with invalid length:", text(len(dataChallenge)), "bytes")
		triggerFirewallException(CHALLENGE, remoteIP)
		return false
	}

	////////////////////////////////////////////////////////////
	// compute our response
	////////////////////////////////////////////////////////////

	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	response := generateHMAC([]byte(GLOBAL.authPassword), dataChallengeArray)
	logDebug(fName, "computed the following response:",
		"\n", base64.StdEncoding.EncodeToString(response))

	////////////////////////////////////////////////////////////
	// create a challenge
	////////////////////////////////////////////////////////////

	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if checkError(err, fName, "error while generating key") {
			conn.Close()
		}
		return key
	}

	randomBytes := newHMACKey()
	logDebug(fName, "generated the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	////////////////////////////////////////////////////////////
	// send the response we computed and the challenge we just created
	////////////////////////////////////////////////////////////

	type outgoingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var outMsgResponse outgoingMessage
	outMsgResponse.Challenge = base64.StdEncoding.EncodeToString(randomBytes[:])
	outMsgResponse.Response = base64.StdEncoding.EncodeToString(response)
	err = conn.WriteJSON(outMsgResponse)
	if checkError(err, fName, "error sending response") {
		return false
	}

	////////////////////////////////////////////////////////////
	// wait for response to challenge
	////////////////////////////////////////////////////////////

	_, remoteHash, err := conn.ReadMessage()
	if checkError(err, fName, "error receiving response") {
		return false
	}

	var remoteHashArray [32]byte
	copy(remoteHashArray[:], randomBytes[:])

	var correctResponse []byte
	correctResponse = generateHMAC([]byte(GLOBAL.authPassword), remoteHashArray)

	if hmac.Equal(correctResponse, remoteHash) != true {
		logError(fName, "initial challenge-response failed",
			"correct:", base64.StdEncoding.EncodeToString(correctResponse),
			"actual:", base64.StdEncoding.EncodeToString(remoteHash))
		triggerFirewallException(CHALLENGE, remoteIP)
		return false
	}

	logDebug(fName, "completed successfully")
	return true
}

func kickoffChallengeResponse(fName string, conn *websocket.Conn) (result bool) {
	fName += " (challenge)"

	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err := io.ReadFull(cryptorand.Reader, key[:])
		checkError(err, fName, "error while generating key")
		return key
	}

	randomBytes := newHMACKey()
	logDebug(fName, "sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	err := conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if checkError(err, fName, "error sending message") {
		return false
	}

	////////////////////////////////////////////////////////////
	// parse incoming response
	////////////////////////////////////////////////////////////

	type incomingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if checkError(err, fName, "error receiving initial response") {
		triggerFirewallException(CHALLENGE, remoteIP)
		return false
	}
	logDebug(fName, "received the following JSON challenge-response:\n", text(inMsg), "\n")

	////////////////////////////////////////////////////////////
	// validate incoming response to our challenge
	////////////////////////////////////////////////////////////

	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	var dataChallengeArray [32]byte
	copy(dataChallengeArray[:], randomBytes[:])

	var correctResponse []byte
	correctResponse = generateHMAC([]byte(GLOBAL.authPassword), dataChallengeArray)

	remoteHash, _ := base64.StdEncoding.DecodeString(inMsg.Response)
	if hmac.Equal(correctResponse, remoteHash) != true {
		logError(fName, "initial challenge-response failed",
			"correct:", base64.StdEncoding.EncodeToString(correctResponse),
			"actual:", base64.StdEncoding.EncodeToString(remoteHash))
		triggerFirewallException(CHALLENGE, remoteIP)
		return false
	}

	////////////////////////////////////////////////////////////
	// respond to challenge sent to us
	////////////////////////////////////////////////////////////

	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	ourResponse := generateHMAC([]byte(GLOBAL.authPassword), remoteChallengeArray)
	logDebug(fName, "local hash for challenge sent by remote end:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if checkError(err, fName, "error sending message to remote end") {
		return false
	}

	logDebug(fName, "completed successfully")
	return true
}
