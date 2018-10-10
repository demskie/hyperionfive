package auth

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"net"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/logging"

	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

// HandleChallengeResponse ...
func HandleChallengeResponse(fName string, conn *websocket.Conn,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) error {

	fName += " (challenge)"

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	err := fw.CheckFirewall(firewall.CHALLENGE, fName, remoteIP)
	if err != nil {
		return err
	}

	_, dataChallenge, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("%v error receiving challenge from %v because %v", fName, remoteIP, err)
	}

	var dataChallengeArray [32]byte

	if len(dataChallenge) == 32 {
		copy(dataChallengeArray[:], dataChallenge)
	} else {
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return fmt.Errorf("%v received a challenge from %v with invalid length of %v bytes",
			fName, remoteIP, len(dataChallenge))
	}

	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	response := generateHMAC([]byte(args.AuthPassword), dataChallengeArray)

	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if log.CheckError(err, fName, "error while generating key") {
			conn.Close()
		}
		return key
	}

	randomBytes := newHMACKey()

	type outgoingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var outMsgResponse outgoingMessage
	outMsgResponse.Challenge = base64.StdEncoding.EncodeToString(randomBytes[:])
	outMsgResponse.Response = base64.StdEncoding.EncodeToString(response)
	err = conn.WriteJSON(outMsgResponse)
	if err != nil {
		return fmt.Errorf("%v error sending response because %v", fName, err)
	}

	_, remoteHash, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("%v error receiving response because %v", fName, err)
	}

	var remoteHashArray [32]byte
	copy(remoteHashArray[:], randomBytes[:])

	var correctResponse []byte
	correctResponse = generateHMAC([]byte(args.AuthPassword), remoteHashArray)

	if hmac.Equal(correctResponse, remoteHash) != true {
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return fmt.Errorf("%v initial challenge-response failed", fName)
	}

	return nil
}

// KickoffChallengeResponse ...
func KickoffChallengeResponse(fName string, conn *websocket.Conn,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) error {

	fName += " (challenge)"

	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err := io.ReadFull(cryptorand.Reader, key[:])
		log.CheckError(err, fName, "error while generating key")
		return key
	}

	randomBytes := newHMACKey()

	err := conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if err != nil {
		return fmt.Errorf("%v error sending message because %v", fName, err)
	}

	type incomingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if err != nil {
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return fmt.Errorf("%v error receiving initial response because %v", fName, err)
	}

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
		fw.TriggerFirewallException(firewall.CHALLENGE, remoteIP)
		return fmt.Errorf("%v initial challenge-response failed", fName)
	}

	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	ourResponse := generateHMAC([]byte(args.AuthPassword), remoteChallengeArray)

	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if err != nil {
		return fmt.Errorf("%v error sending message to remote end %v", fName, err)
	}

	return nil
}
