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
	"regexp"
	"runtime"
	"sync"

	"flag"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

func init() {

	// start logging routine
	logInfoChan = make(chan string)
	logInfo("(initLogInfo) - init completed")
	go handleLogging()

	// initializing global IP blacklist
	clientBlacklist = map[string]*FirewallClientValues{}

	// initializing global IP session count
	clientSessionCount = make(map[string]uint16)

	// define commandline flags to parse for
	galaxyName = *flag.String("galaxyname", "", "specify galaxy name")
	inputFile = *flag.String("input", "", "specify data source, either json or blob")
	outputFile = *flag.String("output", "", "override export destination, either json or blob")
	googleAudience = *flag.String("audience", "", "override google oauth audience")
	authPassword = *flag.String("password", "", "specify password for authentication purposes")
	staticFrontend = *flag.String("staticfrontend", "", "specify static frontend address")
	frontendBool = *flag.Bool("frontend", false, "enable frontend mode")
	backendBool = *flag.Bool("backend", false, "enable backend mode")
	createGalaxyBool = *flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBool = *flag.Bool("debug", false, "enable debug logging")
	singleThreadedBool = *flag.Bool("singlethreaded", false, "force singlethreaded behavior")

	// pull flags attached to CLI invocation
	flag.Parse()

	// handle default and/or invalid parameters
	if googleAudience == "" {
		googleAudience = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}
	if authPassword == "" {
		logInfo("(init) FATALERROR: no password provided")
		waitForLogging.Wait()
		os.Exit(1)
	}
	switch {
	case backendBool == false && frontendBool == false:
		logInfo("(init) FATALERROR: no backend/frontend mode specified")
		waitForLogging.Wait()
		os.Exit(1)
	case backendBool == true && frontendBool == true:
		logInfo("(init) FATALERROR: BOTH backend/frontend mode specified - DoesNotCompute")
		waitForLogging.Wait()
		os.Exit(1)
	}

	// singlethreaded mode for debugging (and emergencies)
	if singleThreadedBool {
		numberOfCores = 1
	} else {
		numberOfCores = runtime.NumCPU()
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	if frontendBool == true {
		startFrontendMode()
		for len(logInfoChan) != 0 {
			time.Sleep(time.Second)
		}
		loggingMutex.Lock()
		fmt.Println("")
		fmt.Println("##############################################")
		fmt.Println("#     Frontend has finished executing!!!     #")
		fmt.Println("##############################################")
		fmt.Println("")
		loggingMutex.Unlock()
		time.Sleep(1 * time.Hour)
		os.Exit(1)
	} else {
		startBackendMode()
		for len(logInfoChan) != 0 {
			time.Sleep(time.Second)
		}
		loggingMutex.Lock()
		fmt.Println("")
		fmt.Println("##############################################")
		fmt.Println("#     Backend has finished executing!!!      #")
		fmt.Println("##############################################")
		fmt.Println("")
		loggingMutex.Unlock()
		time.Sleep(1 * time.Hour)
		os.Exit(1)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func checkFirewall(fName string, client string, check string) bool {

	fName = "(" + fName + ") "

	clientBlacklist[client].RLock()
	_, exists := clientBlacklist[client]

	if exists == true {
		var limit int
		var count uint16
		var lastTime int64
		switch check {
		case "challenge":
			limit = constChallenge
			count = clientBlacklist[client].failedChallengeCount
			lastTime = clientBlacklist[client].failedChallengeTime
		case "googleAuth":
			limit = constGoogleAuth
			count = clientBlacklist[client].failedGoogleAuthCount
			lastTime = clientBlacklist[client].failedGoogleAuthTime
		case "sessionLimit":
			limit = constClientSession
			count = clientBlacklist[client].failedSessionLimitCount
			lastTime = clientBlacklist[client].failedSessionLimitTime
		default:
			logInfo(fName + "checkFirewallHistory was given an invalid 'check' parameter")
			return false
		}
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - lastTime
		blacklistExpGrowth := int64((count) / uint16(limit))
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo(fName+"blacklisted client attempting to connect:", client,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason:", check,
				"\n\tnumberOfOffenses:", string(count))
			return false
		}
		clientBlacklist[client].RUnlock()
	} else {
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = &FirewallClientValues{}
		clientBlacklistMutex.Unlock()
		return true
	}

	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func handleChallengeResponse(fName string, conn *websocket.Conn, clientIP string) (result bool) {

	// string that will be appended to log messages
	fName = "(" + fName + ") "

	////////////////////////////////////////////////////////////
	// receive initial challenge from frontend
	////////////////////////////////////////////////////////////

	_, dataChallenge, err := conn.ReadMessage()
	if checkError(fName+"error receiving challenge from backend:", err) {
		return false
	}
	logDebug(fName+"received the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(dataChallenge))

	////////////////////////////////////////////////////////////
	// ensure that challenge is 256bits and cast to array if so
	////////////////////////////////////////////////////////////

	var dataChallengeArray [32]byte

	if len(dataChallenge) == 32 {
		copy(dataChallengeArray[:], dataChallenge)
	} else {
		logDebug(fName+"received a challenge with invalid length:",
			string(len(dataChallenge)), "bytes")
		currentTime := time.Now().Unix()
		clientBlacklistMutex.RLock()
		clientFirewallValues, exists := clientBlacklist[clientIP]
		clientBlacklistMutex.RUnlock()
		if exists == true {
			clientFirewallValues.Lock()
			clientFirewallValues.failedChallengeCount++
			clientFirewallValues.failedChallengeTime = currentTime
			clientFirewallValues.Unlock()
			return false
		}
	}

	////////////////////////////////////////////////////////////
	// compute our response
	////////////////////////////////////////////////////////////

	// define function to produce symmetric signature using a shared secret key
	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	// generate response hash based on challenge and our secret password
	response := generateHMAC([]byte(authPassword), dataChallengeArray)
	logDebug(fName+"computed the following response:",
		"\n", base64.StdEncoding.EncodeToString(response))

	////////////////////////////////////////////////////////////
	// create a challenge
	////////////////////////////////////////////////////////////

	// define function to generate 256 bits of randomness
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if checkError(fName+"error while generating key", err) {
			conn.Close()
		}
		return key
	}

	// call above function and store value
	randomBytes := newHMACKey()
	logDebug(fName+"generated the following challenge:",
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
	if checkError(fName+"error sending response:", err) {
		return false
	}

	////////////////////////////////////////////////////////////
	// wait for response to challenge
	////////////////////////////////////////////////////////////

	// receive data from backend
	_, remoteHash, err := conn.ReadMessage()
	if checkError(fName+"error receiving response:", err) {
		return false
	}

	// convert data to array
	var remoteHashArray [32]byte
	copy(remoteHashArray[:], randomBytes[:])

	// generate localHash to compare
	var correctResponse []byte
	correctResponse = generateHMAC([]byte(authPassword), remoteHashArray)

	// check for equality
	if hmac.Equal(correctResponse, remoteHash) != true {
		logInfo(fName+"initial challenge-response failed",
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

	// string that will be appended to log messages
	fName = "(" + fName + ") "

	// define function to generate 256 bits of randomness
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err := io.ReadFull(cryptorand.Reader, key[:])
		if checkError(fName+"error while generating key", err) {
			conn.Close()
		}
		return key
	}

	// call above function and store value
	randomBytes := newHMACKey()
	logDebug(fName+"sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	// send random string to frontend
	err := conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if checkError(fName+"error sending message to frontend loadbalancer:", err) {
		return false
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
	if checkError(fName+"error receiving initial response from frontend loadbalancer:", err) {
		return false
	}

	logDebug(fName+"received the following JSON challenge/response:\n",
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
		logInfo(fName+"initial challenge-response failed:",
			"\n", base64.StdEncoding.EncodeToString(correctResponse),
			"\n", base64.StdEncoding.EncodeToString(remoteHash))
		return false
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
	logDebug(fName+"local hash for challenge sent by LoadBalancer:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	// send response to frontend
	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if checkError(fName+"error sending message to frontend loadbalancer:", err) {
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

var serverPriority uint64

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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

var logInfoChan chan string

func handleLogging() {
	for {
		var singleMessage string
		singleMessage = <-logInfoChan

		for i := 0; i < 10000; i++ {
			select {
			case anotherMessage, doesExist := <-logInfoChan:
				if doesExist {
					singleMessage += "\n" + anotherMessage
				}
			default:
				break
			}
		}
		loggingMutex.Lock()
		color.White(singleMessage)
		loggingMutex.Unlock()
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
		logInfoChan <- string(t + " " + message)
	}()
}

func checkError(message string, err error) bool {
	var errorFound bool
	if err != nil {
		errorFound = true
		go func() {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			err := err.Error() // convert err to string
			loggingMutex.Lock()
			color.Red(t + " " + message + err)
			loggingMutex.Unlock()
		}()
	} else {
		errorFound = false
	}
	return errorFound
}

func logDebug(arrayOfStrings ...string) {
	if debugModeBool == true {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		loggingMutex.Lock()
		color.Cyan(t + " " + message)
		loggingMutex.Unlock()
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func growPopulation(currentPopulation float64, foodSupply int32) float64 {
	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	// population if foodSupply == 0
	referencePopulation := float64(10000)

	// maximum % change allowed in single turn (.0005 is standard)
	maxPercentChange := .005

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

func handleInputParameters(fName string) {

	fName = "(" + fName + ") "

	// is the inputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	var doneMatching bool
	if inputFile == "" {
		doneMatching = true
		logDebug(fName + "no inputfilename specified - skipping data import process")
	} else {
		doneMatching = false
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'blob' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*blob$", inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName + "parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln("data.blob not found")
				} else {
					log.Fatalln("Error reading file:", inputFile, err)
				}
			}
			if err := proto.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'json' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*json$", inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName + "parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln(inputFile, "not found")
				} else {
					log.Fatalln("Error reading file:", inputFile, err)
				}
			}
			if err := json.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// if none of the above logic was successful - we have a fatal error on our hands
	if doneMatching == false {
		log.Fatalln(fName + "unknown inputfilename - skipping data import process")
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// planettype
const (
	TERRAN int = iota
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
	SCOUT int = iota
	COLONY
	CARRIER
	BATTLESHIP
	TITAN
)

// solarsystemtype
const (
	BLUEGIANT int = iota
	YELLOWGIANT
	REDGIANT
	WHITEDWARF
	NEUTRONSTAR
	BLACKHOLE
)

// spaceshipstatus
const (
	OPERATIONAL int = iota
	LIGHTLYDAMAGED
	HEAVILYDAMAGED
	SEVERELYDAMAGED
	HANGINGBYATHREAD
)

// galaxytype
const (
	ELLIPTICAL int = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

var (
	numberOfCores  int
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
)

var (
	galaxyName         string
	inputFile          string
	outputFile         string
	googleAudience     string
	authPassword       string
	staticFrontend     string
	frontendBool       bool
	backendBool        bool
	createGalaxyBool   bool
	debugModeBool      bool
	singleThreadedBool bool
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

const (
	constChallenge     = 1
	constGoogleAuth    = 3
	constClientSession = 25
)

var (
	clientBlacklist         map[string]*FirewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex
)

// FirewallClientValues defines values for firewalling
type FirewallClientValues struct {
	sync.RWMutex
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

var (
	frontendData      FrontendDataStruct
	frontendDataMutex sync.RWMutex
)

// FrontendDataStruct defines values for frontendData
type FrontendDataStruct struct {
	ListOfGalaxies []galInfo `json:"ListOfGalaxies"`
	OtherFrontends []string  `json:"OtherFrontends"`
	Message        string    `json:"Message"`
	AnotherMessage string    `json:"AnotherMessage"`
}
type galInfo struct {
	GalaxyName        string   `json:"GalaxyName"`
	GlobalX           int64    `json:"GlobalX"`
	GlobalY           int64    `json:"GlobalY"`
	ClientIDs         []string `json:"ClientIDs"`
	ClusterOfBackends []string `json:"ClusterOfBackends"`
}

// GetGalIndexOf searches listOfGalaxies for a matching galName
func (fd FrontendDataStruct) GetGalIndexOf(s string) (i int) {
	for i := 0; i < len(fd.ListOfGalaxies); i++ {
		if fd.ListOfGalaxies[i].GalaxyName == s {
			return i
		}
	}
	return -1
}

// ByGalaxyName implements sort.Interface for []galInfo based on the GalaxyName field
type ByGalaxyName []galInfo

func (gn ByGalaxyName) Len() int           { return len(gn) }
func (gn ByGalaxyName) Swap(i, j int)      { gn[i], gn[j] = gn[j], gn[i] }
func (gn ByGalaxyName) Less(i, j int) bool { return gn[i].GalaxyName < gn[j].GalaxyName }

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

var (
	backendData      BackendDataStruct
	backendDataMutex sync.RWMutex
)

// BackendDataStruct defines values for ...
type BackendDataStruct struct {
	ClientIDs             []string
	Frontends             []string
	Backends              []string
	HighestPriority       uint64
	SecondHighestPriority uint64
	GlobalX               int64
	GlobalY               int64
}

const (
	connectedBackendsLimit     = 3
	idleConnectedBackendsLimit = 25
	latencyRingLength          = 25
	backendTickrate            = 1 * time.Second
)

var (
	connectedBackends          []Backend
	connectedBackendsMutex     sync.RWMutex
	idleConnectedBackends      []Backend
	idleConnectedBackendsMutex sync.RWMutex
)

// Backend defines values for ...
type Backend struct {
	ServerIP           string
	AdvertisedPriority uint64
	Latency            *ring.Ring
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// IncomingRequest defines values used in clientRequests
type IncomingRequest struct {
	ClientID      string
	ClientMessage string
	ClientValues  []string
	ServerMessage string
}

var (
	sequenceNumber          uint64
	currentGalaxy           Galaxy
	futureGalaxy            Galaxy
	clientRequests          []IncomingRequest
	partitionedSolarSystems []sync.RWMutex
)

var (
	waitForCores sync.WaitGroup
	systemMutex  []sync.RWMutex
	systemIndex  map[string]int
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

const (
	maxNumberOfPlayers            = 10000
	numSystemsPerPlayer           = 100
	totalNumSystems               = maxNumberOfPlayers * numSystemsPerPlayer
	lightyear                     = fakeToRealMultiplier
	minimumDistanceBetweenSystems = lightyear
	galaxyRadius                  = math.MaxInt64
	realGalaxyRadius              = 125000
	fakeToRealMultiplier          = galaxyRadius / realGalaxyRadius
	oneMillion                    = 1000000
	oneBillion                    = 1000000000
	oneTrillion                   = 1000000000000
)

var (
	rtree        *rtreego.Rtree
	rtreeObjects []rtreego.Spatial
	badSystems   [][]int
)

const (
	numDimensions   = 2
	minBranchFactor = 2
	maxBranchFactor = 2
)

// SystemBox ...
type SystemBox struct {
	location rtreego.Point
	index    int
}

// Bounds ...
func (sb *SystemBox) Bounds() *rtreego.Rect {
	tol := 0.01
	return sb.location.ToRect(tol)
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
