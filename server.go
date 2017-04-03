package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"crypto/tls"
	"net"
	"regexp"
	"sync"

	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"google.golang.org/api/oauth2/v2"

	cryptorand "crypto/rand"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// defining various global variables
var (
	inputFilename    string
	outputFilename   string
	googleAudience   string
	authPassword     string
	loadBalanceBool  bool
	createGalaxyBool bool
	debugModeBool    bool

	clientBlacklist         map[string]FirewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex

	randomSeed     *rand.Rand
	currentGalaxy  Galaxy
	futureGalaxy   Galaxy
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
)

// defining global constants for internal firewall
const (
	constChallenge     = 3
	constGoogleAuth    = 3
	constClientSession = 25
)

// FirewallClientValues defines values for firewalling
type FirewallClientValues struct {
	mutex                   sync.RWMutex
	failedChallengeCount    uint16
	failedChallengeTime     int64
	failedGoogleAuthCount   uint16
	failedGoogleAuthTime    int64
	failedSessionLimitCount uint16
	failedSessionLimitTime  int64
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func init() {

	// initializing global IP blacklist
	clientBlacklist = map[string]FirewallClientValues{}

	// initializing global IP session count
	clientSessionCount = make(map[string]uint16)

	// define commandline flags to parse for
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "specify export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "specify google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	loadBalanceBoolPtr := flag.Bool("loadbalance", false, "enable frontend loadbalancer mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")

	// pull flags attached to CLI invocation
	flag.Parse()

	// handle default and/or invalid parameters
	if *googleAudiencePtr == "" {
		*googleAudiencePtr = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}
	if *authPasswordPtr == "" {
		logInfo("(init) FATALERROR: no password provided")
		waitForLogging.Wait()
		os.Exit(1)
	}

	// convert pointers to values and store in global variables
	inputFilename = *inputFilePtr
	outputFilename = *outputFilePtr
	googleAudience = *googleAudiencePtr
	authPassword = *authPasswordPtr
	loadBalanceBool = *loadBalanceBoolPtr
	createGalaxyBool = *createGalaxyBoolPtr
	debugModeBool = *debugModeBoolPtr

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func main() {

	// branch here based on command line invocation
	if loadBalanceBool == true {
		startLoadBalancingMode()
		log.Fatalln("PROGRAM: LoadBalancingMode is terminating... (and wasn't supposed to)")
		waitForLogging.Wait()
		os.Exit(1)
	} else {
		startBackendMode()
		log.Fatalln("PROGRAM: BackendMode is finished executing")
		waitForLogging.Wait()
		os.Exit(1)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startLoadBalancingMode() {

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
	muxSecure.HandleFunc("/loadbalance", clientLoadbalancingWebSocket)
	muxSecure.HandleFunc("/backend", backendFacingWebSocket)

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

func clientLoadbalancingWebSocket(w http.ResponseWriter, r *http.Request) {

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
			logInfo("(lbsocket) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(lbsocket) blacklisted client attempting to connect:", r.RemoteAddr,
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
	if checkError("(lbsocket) upgrade error:", err) == true {
		return
	}
	logDebug("(lbsocket) opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	go func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]++
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
	if checkError("(lbsocket) error receiving message:", err) == true {
		return
	}
	logDebug("(lbsocket) received the following message\n", fmt.Sprintf("%s", inMsg))

	////////////////////////////////////////////////////////////
	// send id_token to google oauth service
	////////////////////////////////////////////////////////////

	successful := false
	var oauth2Service *oauth2.Service // will be used outside of 'for' loop
	for i := 0; i < constGoogleAuth; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if checkError("(lbsocket) error contacting google:", err) == true {
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
	if checkError("(lbsocket) error parsing incoming google response:", err) == true {
		conn.Close()
		return
	}

	rawJSON, err := tokenInfo.MarshalJSON()
	if checkError("(lbsocket) error MarshalJSON'ing google response:", err) == true {
		conn.Close()
		return
	}

	if debugModeBool == true {
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		checkError("(lbsocket) error jsonIndent'ing google response:", err)
		logDebug("(lbsocket) received the following message from google:\n", string(prettyJSON.Bytes()))
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

	logDebug("(lbsocket) closing socket:", string(r.RemoteAddr))
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func backendFacingWebSocket(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(lbsocket) upgrade error:", err) == true {
		return
	}
	defer conn.Close()

	logDebug("(backendsocket) opening socket:", string(r.RemoteAddr))
}

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
		conn, _, err := dialer.Dial("wss://hyperion5.com/", nil)
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
		logDebug("(backend_init) received the following message\n", fmt.Sprintf("%s", inMsg))

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
		if checkError("(backend_init) error sending message to frontend loadbalancer:", err) {
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
	logDebug("(backend_init) succesfully completed handshake with loadbalancer")

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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func absInt64(val int64) int64 {
	if val < 0 {
		val = -val
	}
	return val
}

func commaFormatUint(n uint64) string {
	in := strconv.FormatUint(n, 10)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}
	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func mulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

const mostNegative = -(mostPositive + 1)
const mostPositive = 1<<63 - 1

func signedMulOverflows(a, b int64) bool {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return false
	}
	if a == mostNegative || b == mostNegative {
		return true
	}
	c := a * b
	return c/b != a
}

func checkError(message string, err error) bool {
	waitForLogging.Add(1)
	var errorFound bool
	if err != nil {
		errorFound = true
		go func() {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			err := err.Error() // convert err to string
			loggingMutex.Lock()
			color.Red(t + " " + message + err)
			loggingMutex.Unlock()
			waitForLogging.Done()
		}()
	} else {
		errorFound = false
	}
	return errorFound
}

func logDebug(arrayOfStrings ...string) {
	waitForLogging.Add(1)
	go func() {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		if debugModeBool == true {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			loggingMutex.Lock()
			color.Cyan(t + " " + message)
			loggingMutex.Unlock()
			waitForLogging.Done()
		}
	}()
}

func logInfo(arrayOfStrings ...string) {
	waitForLogging.Add(1)
	go func() {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		loggingMutex.Lock()
		color.White(t + " " + message)
		loggingMutex.Unlock()
		waitForLogging.Done()
	}()
}

func growPopulation(currentPopulation float64, foodSupply int32) float64 {
	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	referencePopulation := float64(10000) // population if foodSupply == 0
	maxPercentChange := .005              // maximum % change allowed in single turn (.0005 is standard)

	N := currentPopulation
	logDebug(commaFormatUint(uint64(N)))

	randomness := referencePopulation * (1 + rnum.NormFloat64()*.1)
	K := randomness * math.Pow(foodSupplyDbl, 3.4)

	logisticPercentChange := (K - N) / K

	switch {
	case logisticPercentChange < -1:
		logisticPercentChange = -1
	case logisticPercentChange > 1:
		logisticPercentChange = 1
	case logisticPercentChange == 0:
		logisticPercentChange = .000001
	}

	currentPopulation = currentPopulation + (N*maxPercentChange)*logisticPercentChange
	return currentPopulation
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// planettype
const (
	TERRAN int32 = iota
	DESERT
	VOLCANIC
	ICE
	BARREN
	FERROUS
	GREENHOUSE
	OCEANIC
	ASTEROID
	DWARF
)

// spaceshiptype
const (
	SCOUT int32 = iota
	COLONY
	CARRIER
	BATTLESHIP
	TITAN
)

// solarsystemtype
const (
	BLUEGIANT int32 = iota
	YELLOWGIANT
	REDGIANT
	WHITEDWARF
	NEUTRONSTAR
	BLACKHOLE
)

// spaceshipstatus
const (
	OPERATIONAL int32 = iota
	LIGHTLYDAMAGED
	HEAVILYDAMAGED
	SEVERELYDAMAGED
	HANGINGBYATHREAD
)

// galaxytype
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
