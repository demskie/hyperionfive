package main

import (
	"bytes"
	"container/ring"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"sync"

	oauth2 "google.golang.org/api/oauth2/v1"

	"flag"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

func main() {
	if GLOBAL.frontendBool == true {
		startFrontendMode() // frontend.go
	} else if GLOBAL.backendBool == true {
		startBackendMode() // backend.go
	}
	for len(GLOBAL.logInfoChan) != 0 {
		time.Sleep(time.Second)
	}
	GLOBAL.loggingMutex.Lock()
	fmt.Println("##############################################")
	fmt.Println("#      Program has finished executing!!!     #")
	fmt.Println("##############################################")
	GLOBAL.loggingMutex.Unlock()
	time.Sleep(5 * time.Minute)
	os.Exit(1)
}

func init() {
	fName := "(init)"

	// sorry not sorry
	GLOBAL = globalVars{}

	GLOBAL.logInfoChan = make(chan string, oneMillion)
	logInfo(fName, "logging initialized")
	go handleLogging()

	GLOBAL.serverPriority = generateRandomUint64()
	logInfo(fName, fmt.Sprintf("serverPriority=%d", GLOBAL.serverPriority))
	GLOBAL.numberOfCores = 0
	GLOBAL.loggingMutex = sync.Mutex{}
	GLOBAL.waitForLogging = sync.WaitGroup{}
	GLOBAL.waitForCores = map[string]*sync.WaitGroup{}

	GLOBAL.clientBlacklist = clientBlacklist{}
	GLOBAL.clientBlacklist.TotalSessionCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeTime = map[string]int64{}
	GLOBAL.clientBlacklist.FailedGoogleAuthCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedGoogleAuthTime = map[string]int64{}
	GLOBAL.clientBlacklist.FailedSessionLimitCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedSessionLimitTime = map[string]int64{}

	GLOBAL.turningDataSignal = make(chan struct{})
	GLOBAL.newTickSignal = make(chan struct{})
	GLOBAL.mutatingDataBlocker = make(chan struct{})
	GLOBAL.allServersReadySignal = make(chan struct{})
	GLOBAL.currentGalaxy = Galaxy{}
	GLOBAL.futureGalaxy = Galaxy{}
	GLOBAL.uniqueRequestID = make(chan int64, 9999)
	go seqNumGenerator(GLOBAL.uniqueRequestID)
	GLOBAL.bufferedRequests = make([]incomingClientRequest, oneMillion)
	GLOBAL.bufferedRequestsMutex = sync.RWMutex{}

	GLOBAL.backendData = backendData{}
	GLOBAL.backendData.Frontends = []string{}
	GLOBAL.backendData.Backends = []string{}
	GLOBAL.backendData.ClientIDs = map[string][]string{}
	GLOBAL.backendData.SystemBytes = map[string][]byte{}
	GLOBAL.backendData.SystemHashes = [16]map[string]map[string][4]byte{}
	GLOBAL.backendData.ParsedPlayerPtrs = map[string][]*SolarSystem{}
	GLOBAL.backendData.Priorities = map[string]uint64{}
	GLOBAL.backendData.Latencies = map[string]*ring.Ring{}
	GLOBAL.backendData.TickTime = 0
	GLOBAL.backendData.NeighborTickTimes = map[string]uint64{}
	GLOBAL.backendData.ReceivedMessageFrom = []string{}
	GLOBAL.backendData.ConfirmedRequestIDs = map[string][]int64{}

	// define commandline flags to parse for
	GLOBAL.galaxyName = *flag.String("galaxyname", "", "specify galaxy name")
	GLOBAL.inputFile = *flag.String("input", "", "specify data source, either json or blob")
	GLOBAL.outputFile = *flag.String("output", "", "override export destination, either json or blob")
	GLOBAL.googleAudience = *flag.String("audience", "", "override google oauth audience")
	GLOBAL.authPassword = *flag.String("password", "", "specify password for authentication purposes")
	GLOBAL.staticFrontend = *flag.String("staticfrontend", "", "specify static frontend address")
	GLOBAL.frontendBool = *flag.Bool("frontend", false, "enable frontend mode")
	GLOBAL.backendBool = *flag.Bool("backend", false, "enable backend mode")
	GLOBAL.createGalaxyBool = *flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	GLOBAL.debugModeBool = *flag.Bool("debug", false, "enable debug logging")
	GLOBAL.singleThreadedBool = *flag.Bool("singlethreaded", false, "force singlethreaded behavior")
	flag.Parse()

	if GLOBAL.googleAudience == "" {
		GLOBAL.googleAudience = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}

	// sanity check that a password was provided
	if GLOBAL.authPassword == "" {
		logError(fName, "no password provided")
		GLOBAL.waitForLogging.Wait()
		os.Exit(1)
	}

	// sanity check operating mode specified via Booleans
	switch {
	case GLOBAL.backendBool == false && GLOBAL.frontendBool == false:
		logError(fName, "neither 'backend' or 'frontend' operating mode was specified")
		os.Exit(1)
	case GLOBAL.backendBool == true && GLOBAL.frontendBool == true:
		logError(fName, "BOTH backend/frontend mode specified. DoesNotCompute")
		os.Exit(1)
	}

	// singlethreaded mode for debugging (and emergencies)
	if GLOBAL.singleThreadedBool {
		GLOBAL.numberOfCores = 1
	} else {
		GLOBAL.numberOfCores = runtime.NumCPU()
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func checkFirewall(fName string, remoteIPAddr string, check string) bool {

	var limit int
	var count uint16
	var lastTime int64
	var alpha, beta, charlie, delta, echo, foxtrot bool

	GLOBAL.clientBlacklist.RLock()
	switch check {
	case "challenge":
		limit = constChallenge
		count, alpha = GLOBAL.clientBlacklist.FailedChallengeCount[remoteIPAddr]
		lastTime, beta = GLOBAL.clientBlacklist.FailedChallengeTime[remoteIPAddr]
	case "googleAuth":
		limit = constGoogleAuth
		count, charlie = GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIPAddr]
		lastTime, delta = GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIPAddr]
	case "sessionLimit":
		limit = constClientSession
		count, echo = GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIPAddr]
		lastTime, foxtrot = GLOBAL.clientBlacklist.FailedSessionLimitTime[remoteIPAddr]
	default:
		logError(fName, "> gave checkFirewallHistory an invalid 'check' parameter")
		GLOBAL.clientBlacklist.RUnlock()
		// fail open
		return true
	}
	GLOBAL.clientBlacklist.RUnlock()

	currentTime := time.Now().Unix()
	timeSinceBlacklist := currentTime - lastTime
	blacklistExpGrowth := int64((count) / uint16(limit))
	currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
	if timeSinceBlacklist < currentBlacklistGiven {
		logError(fName, "blacklisted client attempting to connect:", remoteIPAddr,
			"\n\tblacklistedFor:", string(timeSinceBlacklist),
			"\n\treason:", check,
			"\n\tnumberOfOffenses:", string(count))
		return false
	}

	if !alpha || !beta || !charlie || !delta || !echo || !foxtrot {
		GLOBAL.clientBlacklist.Lock()
		if !alpha {
			GLOBAL.clientBlacklist.FailedChallengeCount[remoteIPAddr] = 0
		}
		if !beta {
			GLOBAL.clientBlacklist.FailedChallengeTime[remoteIPAddr] = math.MinInt64
		}
		if !charlie {
			GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIPAddr] = 0
		}
		if !delta {
			GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIPAddr] = math.MinInt64
		}
		if !echo {
			GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIPAddr] = 0
		}
		if !foxtrot {
			GLOBAL.clientBlacklist.FailedSessionLimitTime[remoteIPAddr] = math.MinInt64
		}
		GLOBAL.clientBlacklist.Unlock()
	}

	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleGoogleAuth(fName string, conn *websocket.Conn, remoteIPAddr string) (userid string) {

	// this MUST be checked on the other side
	userid = ""

	// receive JSON message from client
	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}

	var inMsg incomingMessage
	err := conn.ReadJSON(&inMsg)
	if checkError(err, fName, "error receiving message") {
		return userid
	}
	logDebug(fName, "received the following message\n", fmt.Sprintf("%s", inMsg))

	////////////////////////////////////////////////////////////
	// send id_token to google oauth service
	////////////////////////////////////////////////////////////

	successful := false
	var oauth2Service *oauth2.Service
	for i := 0; i < constGoogleAuth; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if checkError(err, fName, "error contacting google") {
			continue
		} else {
			successful = true
			break
		}
	}
	if successful == false {
		return userid
	}

	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.IDToken)
	tokenInfo, err := tokenInfoCall.Do()
	if checkError(err, fName, "error parsing incoming google response") {
		return userid
	}

	triggerFailState := func() {
		currentTime := time.Now().Unix()
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIPAddr]++
		GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIPAddr] = currentTime
		GLOBAL.clientBlacklist.Unlock()
	}
	if tokenInfo.VerifiedEmail != true {
		logError(fName, "tokenInfo.VerifiedEmail is not 'true'", remoteIPAddr)
		triggerFailState()
		return userid
	}
	if tokenInfo.Audience != GLOBAL.googleAudience {
		aMessage := fmt.Sprintf("tokenInfo.Audience (%s) does not match googleAudience (%s) remoteIP=%s",
			tokenInfo.Audience, GLOBAL.googleAudience, remoteIPAddr)
		logError(fName, aMessage)
		triggerFailState()
		return userid
	}

	if GLOBAL.debugModeBool == true {
		rawJSON, err := tokenInfo.MarshalJSON()
		checkError(err, fName, "error MarshalJSON'ing google response")
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		checkError(err, fName, "error jsonIndenting google response")
		logDebug(fName, "received the following message from google:\n",
			string(prettyJSON.Bytes()))
	}

	userid = tokenInfo.UserId
	return userid

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleChallengeResponse(fName string, conn *websocket.Conn, clientIP string) (result bool) {

	// receive initial challenge from frontend
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
		logError(fName, "received a challenge with invalid length:",
			string(len(dataChallenge)), "bytes")
		currentTime := time.Now().Unix()
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.FailedChallengeCount[clientIP]++
		GLOBAL.clientBlacklist.FailedChallengeTime[clientIP] = currentTime
		GLOBAL.clientBlacklist.Unlock()
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
			"\n", base64.StdEncoding.EncodeToString(correctResponse),
			"\n", base64.StdEncoding.EncodeToString(remoteHash))
		return false
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffChallengeResponse(fName string, conn *websocket.Conn) (result bool) {

	// create a challenge
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err := io.ReadFull(cryptorand.Reader, key[:])
		if checkError(err, fName, "error while generating key") {
			conn.Close()
		}
		return key
	}

	randomBytes := newHMACKey()
	logDebug(fName, "sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	err := conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if checkError(err, fName, "error sending message to frontend loadbalancer") {
		return false
	}

	////////////////////////////////////////////////////////////
	// parse incoming response
	////////////////////////////////////////////////////////////

	type incomingMessage struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if checkError(err, fName, "error receiving initial response from frontend loadbalancer") {
		return false
	}

	logDebug(fName, "received the following JSON challenge/response:\n",
		fmt.Sprintf("%s\n", inMsg))

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
		logError(fName, "initial challenge-response failed:",
			"\n", base64.StdEncoding.EncodeToString(correctResponse),
			"\n", base64.StdEncoding.EncodeToString(remoteHash))
		return false
	}

	////////////////////////////////////////////////////////////
	// respond to challenge sent to us
	////////////////////////////////////////////////////////////

	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	ourResponse := generateHMAC([]byte(GLOBAL.authPassword), remoteChallengeArray)
	logDebug(fName, "local hash for challenge sent by LoadBalancer:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if checkError(err, fName, "error sending message to frontend loadbalancer") {
		return false
	}

	return true
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

func appendStringIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func removeStringIfPresent(slice []string, s string) []string {
	for i, ele := range slice {
		if ele == s {
			slice[i] = slice[len(slice)-1]
			slice = slice[:len(slice)-1]
		}
	}
	return slice
}

func generateRandomUint64() uint64 {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	rnum := rand.New(seed())
	return rnum.Uint64()
}

func getCryptoRnum() *rand.Rand {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	return rand.New(seed())
}

func getPosOrNegOne() int {
	var randomNum int8
	b := make([]byte, 1)
	cryptorand.Read(b)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &randomNum)
	if randomNum >= 0 {
		return 1
	}
	return -1
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleLogging() {
	for {
		var singleMessage string
		singleMessage = <-GLOBAL.logInfoChan
		for i := 0; i < 10000; i++ {
			select {
			case anotherMessage, doesExist := <-GLOBAL.logInfoChan:
				if doesExist {
					singleMessage += "\n" + anotherMessage
				}
			default:
				break
			}
		}
		GLOBAL.loggingMutex.Lock()
		color.White(singleMessage)
		GLOBAL.loggingMutex.Unlock()
	}
}

func logInfo(arrayOfStrings ...string) {
	var message string
	for i := range arrayOfStrings {
		if i > 0 {
			message += " "
		}
		message += arrayOfStrings[i]
	}
	go func() {
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		GLOBAL.logInfoChan <- string(t + " " + message)
	}()
}

func logDebug(arrayOfStrings ...string) {
	if GLOBAL.debugModeBool == true {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		GLOBAL.loggingMutex.Lock()
		color.Cyan("DEBUG:" + t + message)
		GLOBAL.loggingMutex.Unlock()
	}
}

func checkError(err error, arrayOfStrings ...string) bool {
	var errorFound bool
	if err != nil {
		errorFound = true
		go func() {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			var message string
			for i := range arrayOfStrings {
				if i > 0 {
					message += " "
				}
				message += arrayOfStrings[i]
			}
			err := err.Error() // convert err to string
			GLOBAL.loggingMutex.Lock()
			color.Red("ERROR: " + t + " " + message + " > " + err)
			GLOBAL.loggingMutex.Unlock()
		}()
	} else {
		errorFound = false
	}
	return errorFound
}

func logError(arrayOfStrings ...string) {
	var message string
	for i := range arrayOfStrings {
		if i > 0 {
			message += " "
		}
		message += arrayOfStrings[i]
	}
	go func() {
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		GLOBAL.loggingMutex.Lock()
		color.Red("ERROR: " + t + " " + message)
		GLOBAL.loggingMutex.Unlock()
	}()
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func seqNumGenerator(randNumChan chan int64) {
	var number int64
	b := make([]byte, 8)
	cryptorand.Read(b)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &number)
	for {
		randNumChan <- number
		number++
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleInputParameters(fName string) {

	// is the inputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	var doneMatching bool
	if GLOBAL.inputFile == "" {
		doneMatching = true
		logDebug(fName, "no inputfilename specified - skipping data import process")
	} else {
		doneMatching = false
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'blob' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*blob$", GLOBAL.inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName, "parsing blob inputfilename")
			in, err := ioutil.ReadFile(GLOBAL.inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln("data.blob not found")
				} else {
					log.Fatalln("Error reading file:", GLOBAL.inputFile, err)
				}
			}
			if err := proto.Unmarshal(in, &GLOBAL.currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'json' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*json$", GLOBAL.inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName, "parsing blob inputfilename")
			in, err := ioutil.ReadFile(GLOBAL.inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln(GLOBAL.inputFile, "not found")
				} else {
					log.Fatalln("Error reading file:", GLOBAL.inputFile, err)
				}
			}
			if err := json.Unmarshal(in, &GLOBAL.currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// if none of the above logic was successful - we have a fatal error on our hands
	if doneMatching == false {
		log.Fatalln(fName, "unknown inputfilename. skipping data import process")
	}
}
