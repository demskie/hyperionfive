package auth

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hyperionfive/arguments"
	"hyperionfive/firewall"
	"hyperionfive/logging"
	"io"
	"net"

	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

// HandleChallengeResponse ...
func HandleChallengeResponse(fName string, conn *websocket.Conn,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (result bool) {

	fName += " (challenge)"

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	err := fw.CheckFirewall(firewall.CHALLENGE, fName, remoteIP)
	if log.CheckError(err) {
		return false
	}

	// receive initial challenge from remote end
	_, dataChallenge, err := conn.ReadMessage()
	if log.CheckError(err, fName, "error receiving challenge from backend") {
		return false
	}
	log.Debug(fName, "received the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(dataChallenge))

	// ensure that challenge is 256bits and cast to array if so
	var dataChallengeArray [32]byte

	if len(dataChallenge) == 32 {
		copy(dataChallengeArray[:], dataChallenge)
	} else {
		log.Error(fName, fmt.Sprintf("received a challenge with invalid length: %v bytes",
			len(dataChallenge)))
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return false
	}

	// compute our response
	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	response := generateHMAC([]byte(args.AuthPassword), dataChallengeArray)
	log.Debug(fName, "computed the following response:",
		"\n", base64.StdEncoding.EncodeToString(response))

	// create a challenge
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if log.CheckError(err, fName, "error while generating key") {
			conn.Close()
		}
		return key
	}

	randomBytes := newHMACKey()
	log.Debug(fName, "generated the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	// send the response we computed and the challenge we just created
	type outgoingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var outMsgResponse outgoingMessage
	outMsgResponse.Challenge = base64.StdEncoding.EncodeToString(randomBytes[:])
	outMsgResponse.Response = base64.StdEncoding.EncodeToString(response)
	err = conn.WriteJSON(outMsgResponse)
	if log.CheckError(err, fName, "error sending response") {
		return false
	}

	// wait for response to challenge
	_, remoteHash, err := conn.ReadMessage()
	if log.CheckError(err, fName, "error receiving response") {
		return false
	}

	var remoteHashArray [32]byte
	copy(remoteHashArray[:], randomBytes[:])

	var correctResponse []byte
	correctResponse = generateHMAC([]byte(args.AuthPassword), remoteHashArray)

	if hmac.Equal(correctResponse, remoteHash) != true {
		log.Error(fName, "initial challenge-response failed",
			"correct:", base64.StdEncoding.EncodeToString(correctResponse),
			"actual:", base64.StdEncoding.EncodeToString(remoteHash))
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return false
	}

	log.Debug(fName, "completed successfully")
	return true
}

// KickoffChallengeResponse ...
func KickoffChallengeResponse(fName string, conn *websocket.Conn,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (result bool) {

	fName += " (challenge)"

	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err := io.ReadFull(cryptorand.Reader, key[:])
		log.CheckError(err, fName, "error while generating key")
		return key
	}

	randomBytes := newHMACKey()
	log.Debug(fName, "sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	err := conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if log.CheckError(err, fName, "error sending message") {
		return false
	}

	// parse incoming response
	type incomingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if log.CheckError(err, fName, "error receiving initial response") {
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return false
	}
	log.Debug(fName, fmt.Sprintf("received the following JSON challenge-response:\n%v\n", inMsg))

	// validate incoming response to our challenge
	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	var dataChallengeArray [32]byte
	copy(dataChallengeArray[:], randomBytes[:])

	var correctResponse []byte
	correctResponse = generateHMAC([]byte(args.AuthPassword), dataChallengeArray)

	remoteHash, _ := base64.StdEncoding.DecodeString(inMsg.Response)
	if hmac.Equal(correctResponse, remoteHash) != true {
		log.Error(fName, "initial challenge-response failed",
			"correct:", base64.StdEncoding.EncodeToString(correctResponse),
			"actual:", base64.StdEncoding.EncodeToString(remoteHash))
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return false
	}

	// respond to challenge sent to us
	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	ourResponse := generateHMAC([]byte(args.AuthPassword), remoteChallengeArray)
	log.Debug(fName, "local hash for challenge sent by remote end:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if log.CheckError(err, fName, "error sending message to remote end") {
		return false
	}

	log.Debug(fName, "completed successfully")
	return true
}
