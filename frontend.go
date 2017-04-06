package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v1"

	cryptorand "crypto/rand"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startFrontendMode() {

	// setup httpMux to avoid leaking default handlers
	muxSecure := http.NewServeMux()

	// to enable perfect forward secrecy it's necessary to add 'Strict-Transport-Security'
	// and since we are not writing out own FileServer, we'll need to add Headers this way
	addSecurityHeaders := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.ServeHTTP(w, r)
		}
	}
	directory := http.Dir("D:\\OneDrive\\GoPrototypes\\src\\GOSERVER\\websiteroot\\")
	fileserverWithModifiedHeaders := addSecurityHeaders(http.FileServer(directory))

	// setup the fileserver handler now that the necessary security headers have been added
	muxSecure.Handle("/", fileserverWithModifiedHeaders)

	// setup the websocket handlers
	muxSecure.HandleFunc("/loadbalance", loadbalanceClients)
	muxSecure.HandleFunc("/backend", handshakeWithBackend)

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         ":443",
		Handler:      muxSecure,

		// defining various security parameters used to specify accepted ciphersuites
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},

		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// define waitgroup to be used to stop premature termination
	var lbWaitGroup sync.WaitGroup

	// serve HTTPS requests with config defined above
	lbWaitGroup.Add(1)
	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			waitForLogging.Wait()
			os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// define nonsecure HTTP parameters
	srvInsecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			url := "https://" + req.Host + req.URL.String()
			http.Redirect(w, req, url, http.StatusMovedPermanently)
		}),
	}

	// serve HTTP requests by redirecting to HTTPS
	lbWaitGroup.Add(1)
	go func() {
		errServeInsecure := srvInsecure.ListenAndServe()
		if checkError("(ListenAndServeInsecure) Something went wrong:", errServeInsecure) {
			waitForLogging.Wait()
			os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// use waitgroup to stop permature termination unless all the above goroutines are finished executing
	lbWaitGroup.Wait()
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func loadbalanceClients(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	// parsing IP address of this new client
	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	clientBlacklistMutex.RLock()
	clientFirewallValues, exists := clientBlacklist[client]
	clientBlacklistMutex.RUnlock()
	if exists == true {
		clientFirewallValues.mutex.RLock()
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - clientFirewallValues.failedChallengeTime
		blacklistExpGrowth := int64((clientFirewallValues.failedChallengeCount) / constChallenge)
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(loadbalanceClients) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(loadbalanceClients) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedGoogleAuth",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedGoogleAuthCount))
			return
		}
		clientFirewallValues.mutex.RUnlock()
	} else {
		// intialize client with default values
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = FirewallClientValues{}
		clientBlacklistMutex.Unlock()
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	// defining websocket.Upgrader parameters
	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	// attempting to process the incoming client's request
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(loadbalanceClients) upgrade error:", err) == true {
		return
	}
	logDebug("(loadbalanceClients) opening socket:", string(r.RemoteAddr))

	// enable write compression

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	go func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]++
		} else {
			clientSessionCount[client] = 1
		}
		clientSessionCountMutex.Unlock()
	}()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive JSON message from client
	////////////////////////////////////////////////////////////

	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if checkError("(loadbalanceClients) error receiving message:", err) == true {
		return
	}
	logDebug("(loadbalanceClients) received the following message\n", fmt.Sprintf("%s", inMsg))

	////////////////////////////////////////////////////////////
	// send id_token to google oauth service
	////////////////////////////////////////////////////////////

	successful := false
	var oauth2Service *oauth2.Service // will be used outside of 'for' loop
	for i := 0; i < constGoogleAuth; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if checkError("(loadbalanceClients) error contacting google:", err) == true {
			continue
		} else {
			successful = true
			break
		}
	}
	if successful == false {
		currentTime := time.Now().Unix()
		clientFirewallValues.mutex.Lock()
		clientFirewallValues.failedGoogleAuthCount += constGoogleAuth
		clientFirewallValues.failedGoogleAuthTime = currentTime
		clientFirewallValues.mutex.Unlock()
		conn.Close()
		return
	}

	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.IDToken)
	tokenInfo, err := tokenInfoCall.Do()
	if checkError("(loadbalanceClients) error parsing incoming google response:", err) == true {
		conn.Close()
		return
	}

	rawJSON, err := tokenInfo.MarshalJSON()
	if checkError("(loadbalanceClients) error MarshalJSON'ing google response:", err) == true {
		conn.Close()
		return
	}

	if debugModeBool == true {
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		checkError("(loadbalanceClients) error jsonIndent'ing google response:", err)
		logDebug("(loadbalanceClients) received the following message from google:\n", string(prettyJSON.Bytes()))
	}

	////////////////////////////////////////////////////////////
	// send list of galaxy servers to client
	////////////////////////////////////////////////////////////

	type PotentialServers struct {
		GalaxyName string
		ServerList []string
	}
	type outgoingMessage struct {
		Servers []PotentialServers
	}

	////////////////////////////////////////////////////////////
	// close websocket with completion
	////////////////////////////////////////////////////////////

	logDebug("(loadbalanceClients) closing socket:", string(r.RemoteAddr))
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handshakeWithBackend(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	// parsing IP address of this new client
	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	clientBlacklistMutex.RLock()
	clientFirewallValues, exists := clientBlacklist[client]
	clientBlacklistMutex.RUnlock()
	if exists == true {
		clientFirewallValues.mutex.RLock()
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - clientFirewallValues.failedChallengeTime
		blacklistExpGrowth := int64((clientFirewallValues.failedChallengeCount) / constChallenge)
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeWithBackend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeWithBackend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedGoogleAuth",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedGoogleAuthCount))
			return
		}
		clientFirewallValues.mutex.RUnlock()
	} else {
		// intialize client with default values
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = FirewallClientValues{}
		clientBlacklistMutex.Unlock()
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	if checkError("(handshakeWithBackend) upgrade error:", err) == true {
		return
	}
	logDebug("(handshakeWithBackend) opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// modify sessionCount accordingly
	////////////////////////////////////////////////////////////

	go func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]++
		} else {
			clientSessionCount[client] = 1
			logInfo("(handshakeWithBackend) unsuccessfully tried to increment",
				"'clientSessionCount' for this client:", client)
		}
		clientSessionCountMutex.Unlock()
	}()

	defer func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]--
		} else {
			clientSessionCount[client] = 0
			logInfo("(handshakeWithBackend) unsuccessfully tried to decrement",
				"'clientSessionCount' for this client:", client)
		}
		clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive initial challenge from backend
	////////////////////////////////////////////////////////////

	_, dataChallenge, err := conn.ReadMessage()
	if checkError("(handshakeWithBackend) error receiving challenge from backend:", err) == true {
		return
	}
	if len(dataChallenge) != 256 {
		logDebug("(handshakeWithBackend) received a challenge with invalid length:", string(len(dataChallenge)), "bytes")
		currentTime := time.Now().Unix()
		clientFirewallValues.mutex.Lock()
		clientFirewallValues.failedChallengeCount++
		clientFirewallValues.failedChallengeTime = currentTime
		clientFirewallValues.mutex.Unlock()
		return
	}
	logDebug("(handshakeWithBackend) received the following challenge:\n", fmt.Sprintf("%s", dataChallenge))

	////////////////////////////////////////////////////////////
	// compute response and create a challenge of our own
	////////////////////////////////////////////////////////////

	// create local hash to compare with Response
	hash := hmac.New(sha512.New512_256, []byte(authPassword))
	hash.Write(dataChallenge)
	expectedHash := hash.Sum(nil)
	logDebug("(handshakeWithBackend) computed the following response:\n", fmt.Sprintf("%s", expectedHash))

	// generate 256 random bytes
	randomBytes := make([]byte, 256)
	_, err = cryptorand.Read(randomBytes)
	if checkError("(handshakeWithBackend) error while generating bytes:", err) {
		return
	}
	logDebug("(handshakeWithBackend) generated the following challenge:\n", fmt.Sprintf("%s", randomBytes))

	////////////////////////////////////////////////////////////
	// send the response in addition to own challenge
	////////////////////////////////////////////////////////////

	type outgoingMessage struct {
		Response  string `json:"response"`
		Challenge string `json:"challenge"`
	}

	var outMsg outgoingMessage
	outMsg.Response = base64.StdEncoding.EncodeToString(expectedHash)
	outMsg.Challenge = base64.StdEncoding.EncodeToString(randomBytes)
	err = conn.WriteJSON(outMsg)
	if checkError("(handshakeWithBackend) error sending response to backend:", err) == true {
		return
	}

	////////////////////////////////////////////////////////////
	// wait for response to challenge
	////////////////////////////////////////////////////////////

	// receive data from backend
	_, remoteHash, err := conn.ReadMessage()
	if checkError("(handshakeWithBackend) error receiving response from backend:", err) == true {
		return
	}

	// check for equality
	if hmac.Equal(expectedHash, remoteHash) != true {
		logInfo("(handshakeWithBackend) initial challenge-response failed:")
		return
	}

	////////////////////////////////////////////////////////////
	// say hello
	////////////////////////////////////////////////////////////

	// send response to frontend
	err = conn.WriteMessage(websocket.TextMessage, []byte("hellothere"))
	if checkError("(handshakeWithBackend) error sending hellothere to backend:", err) {
		return
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
