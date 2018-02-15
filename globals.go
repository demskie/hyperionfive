package main

import (
	"flag"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"
)

// GLOBAL declaration
var GLOBAL globalVars

type globalVars struct {
	galaxyName         string
	inputFile          string
	outputFile         string
	googleAudience     string
	authPassword       string
	staticFrontend     string
	localInterface     string
	localAddress       string
	enableFrontendBool bool
	enableBackendBool  bool
	createGalaxyBool   bool
	debugModeBool      bool
	singleThreadedBool bool
	testModeBool       bool

	exitProgram chan struct{}

	logInfoChan chan loggingMessage

	serverPriority  uint64
	numberOfCores   int
	debugMultiplier int

	clientBlacklist clientBlacklist

	backendData           backendData
	galaxy                *Galaxy
	galaxyMutex           sync.RWMutex
	turningDataSignal     chan struct{}
	backendSignal         chan struct{}
	clientSignal          chan struct{}
	allServersReadySignal chan struct{}
	allRequests           allRequests
	uniqueRequestID       chan int64

	frontendData frontendData
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type clientBlacklist struct {
	sync.RWMutex
	LiveConnections         map[string][]string
	FailedChallengeCount    map[string]uint16
	FailedChallengeTime     map[string]time.Time
	FailedGoogleAuthCount   map[string]uint16
	FailedGoogleAuthTime    map[string]time.Time
	FailedSessionLimitCount map[string]uint16
	FailedSessionLimitTime  map[string]time.Time
}

type allRequests struct {
	sync.Mutex
	Buffered     map[int64]modifiedClientRequest  // uniqueID : clientRequest
	ConfirmedIDs map[string]map[int64]emptyStruct // neighborIP : uniqueID : struct{}{}
}

type originalClientRequest struct {
	ClientMessage string   `json:"clientMessage"`
	ClientValues  []string `json:"clientValues"`
}

type modifiedClientRequest struct {
	ClientID      string
	ClientMessage string
	ClientValues  []string
	ReceivedTime  time.Time
}

type backendData struct {
	sync.RWMutex
	GalaxyName              string
	GlobalX                 int64
	GlobalY                 int64
	Frontends               []string
	Backends                []string
	SystemClientIDs         map[string][]string
	SystemBytes             map[string][]byte           // sysName : sysBytes
	SystemsInRecovery       map[string]bool             // sysName : hasReceivedData
	NeighborRecoveryMapping map[string][]string         // neighbor : sysNameSlice
	SystemHashes            map[string]int64            // sysName : sysHash
	NeighborSystemHashes    map[string]map[string]int64 // neighbor : sysName : sysHash
	ParsedPlayerPtrs        map[string][]*SolarSystem   // playerName : solarSystemPointers
	LocalTickTime           time.Duration
	NeighborTickTimes       map[string]time.Duration
	ReceivedMessageFrom     []string
}

type frontendData struct {
	sync.RWMutex
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	BackendLoad    map[string]int
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type emptyStruct struct{}

type backendToBackendMessage struct {
	Frontends         []string                        //
	Backends          []string                        //
	Priority          uint64                          //
	SystemHashes      map[string]int64                //
	BufferedRequests  map[int64]modifiedClientRequest // uniqueID : clientRequest
	SystemBytes       map[string][]byte               //
	SystemsInRecovery []string                        //
	SystemOrder       []string                        //
	TickTime          time.Duration                   //
	NumberOfCores     int                             // not implemented
}

type frontendToFrontendMessage struct {
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

type backendToFrontendMessage struct {
	GalaxyName     string
	GlobalX        int64
	GlobalY        int64
	ClientIDs      []string
	OtherBackends  []string
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

type frontendToBackendMessage struct {
	GalaxyName     string
	GlobalX        int64
	GlobalY        int64
	OtherBackends  []string
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

const (
	constChallenge     = 1
	constGoogleAuth    = 3
	constClientSession = 25
)

const (
	numDimensions   = 2
	minBranchFactor = 2
	maxBranchFactor = 2
)

const (
	maxNumberOfPlayers            = 10000
	numSystemsPerPlayer           = 10
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

const (
	diediedie      = "die_die_die"
	sameaslasttime = "same_as_last_time"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	const fName = "(init)"

	GLOBAL = globalVars{}

	GLOBAL.exitProgram = make(chan struct{})

	GLOBAL.logInfoChan = make(chan loggingMessage, oneMillion)
	logInfo(fName, "logging initialized")
	go handleLogging()

	GLOBAL.serverPriority = generateRandomUint64()
	logInfo(fName, "local serverPriority is", commaFormatUint(uint64(GLOBAL.serverPriority)))
	GLOBAL.numberOfCores = 0
	GLOBAL.debugMultiplier = 0

	GLOBAL.clientBlacklist = clientBlacklist{}
	GLOBAL.clientBlacklist.LiveConnections = map[string][]string{} // remoteIP : fName
	GLOBAL.clientBlacklist.FailedChallengeCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedGoogleAuthCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedGoogleAuthTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedSessionLimitCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedSessionLimitTime = map[string]time.Time{}

	GLOBAL.backendData = backendData{}
	GLOBAL.backendData.GalaxyName = ""
	GLOBAL.backendData.GlobalX = 0
	GLOBAL.backendData.GlobalY = 0
	GLOBAL.backendData.Frontends = []string{}
	GLOBAL.backendData.Backends = []string{}
	GLOBAL.backendData.SystemClientIDs = map[string][]string{}
	GLOBAL.backendData.SystemBytes = map[string][]byte{}
	GLOBAL.backendData.SystemsInRecovery = map[string]bool{}
	GLOBAL.backendData.NeighborRecoveryMapping = map[string][]string{}
	GLOBAL.backendData.SystemHashes = map[string]int64{}
	GLOBAL.backendData.NeighborSystemHashes = map[string]map[string]int64{}
	GLOBAL.backendData.ParsedPlayerPtrs = map[string][]*SolarSystem{}
	GLOBAL.backendData.LocalTickTime = time.Duration(math.MinInt64)
	GLOBAL.backendData.NeighborTickTimes = map[string]time.Duration{}
	GLOBAL.backendData.ReceivedMessageFrom = []string{}
	GLOBAL.galaxy = &Galaxy{}
	GLOBAL.galaxyMutex = sync.RWMutex{}
	GLOBAL.turningDataSignal = make(chan struct{})
	GLOBAL.backendSignal = make(chan struct{})
	GLOBAL.clientSignal = make(chan struct{})
	GLOBAL.allServersReadySignal = make(chan struct{})
	GLOBAL.allRequests = allRequests{}
	GLOBAL.allRequests.Buffered = map[int64]modifiedClientRequest{}
	GLOBAL.allRequests.ConfirmedIDs = map[string]map[int64]emptyStruct{}
	GLOBAL.uniqueRequestID = make(chan int64, 9999)

	GLOBAL.frontendData = frontendData{}
	GLOBAL.frontendData.GalGlobalX = map[string]int64{}
	GLOBAL.frontendData.GalGlobalY = map[string]int64{}
	GLOBAL.frontendData.GalClientIDs = map[string][]string{}
	GLOBAL.frontendData.GalBackends = map[string][]string{}
	GLOBAL.frontendData.BackendLoad = map[string]int{}
	GLOBAL.frontendData.OtherFrontends = []string{}
	GLOBAL.frontendData.Message = ""
	GLOBAL.frontendData.AnotherMessage = ""

	// define commandline flags to parse
	galaxyNamePtr := flag.String("galaxyname", "", "specify galaxy name")
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "override export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "override google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	staticFrontendPtr := flag.String("staticfrontend", "", "specify static frontend address")
	localInterfacePtr := flag.String("interface", "", "override local interface to use")
	localAddressPtr := flag.String("localip", "", "override local ip to use")
	enableFrontendBoolPtr := flag.Bool("frontend", false, "enable frontend mode")
	enableBackendBoolPtr := flag.Bool("backend", false, "enable backend mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")
	singleThreadedBoolPtr := flag.Bool("singlethreaded", false, "force singlethreaded behavior")
	testModeBoolPtr := flag.Bool("test", false, "enable testing mode")

	// parse the arguments
	flag.Parse()
	GLOBAL.galaxyName = *galaxyNamePtr
	GLOBAL.inputFile = *inputFilePtr
	GLOBAL.outputFile = *outputFilePtr
	GLOBAL.googleAudience = *googleAudiencePtr
	GLOBAL.authPassword = *authPasswordPtr
	GLOBAL.staticFrontend = *staticFrontendPtr
	GLOBAL.localInterface = *localInterfacePtr
	GLOBAL.localAddress = *localAddressPtr
	GLOBAL.enableFrontendBool = *enableFrontendBoolPtr
	GLOBAL.enableBackendBool = *enableBackendBoolPtr
	GLOBAL.createGalaxyBool = *createGalaxyBoolPtr
	GLOBAL.debugModeBool = *debugModeBoolPtr
	GLOBAL.singleThreadedBool = *singleThreadedBoolPtr
	GLOBAL.testModeBool = *testModeBoolPtr

	if GLOBAL.googleAudience == "" {
		GLOBAL.googleAudience = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}

	// sanity check that a password was provided
	if GLOBAL.authPassword == "" {
		logInfo(fName, "no password provided, using default password")
		GLOBAL.authPassword += "uVXMHpASbDngPe41pUP1"
		GLOBAL.authPassword += "CKwRCU6ZGap6IxQ6OQaM"
		GLOBAL.authPassword += "AtJgeaIhHLzO0IibxE8B"
		GLOBAL.authPassword += "7aYOpAXkC388SRg4G3jO"
		GLOBAL.authPassword += "kkgrBnOOUPxnT6QopsFH"
		GLOBAL.authPassword += "a0F7qFScNxYuCaiRvii3"
		GLOBAL.authPassword += "WvhhNNsFaRupwZc7SXXw"
		GLOBAL.authPassword += "cPk93TWZs0kllqVhCN0G"
		GLOBAL.authPassword += "9oKiFpa0p7Mfwd8CqHiz"
		GLOBAL.authPassword += "3ghjZw6FI17DjdObCXSF"
	}

	// singlethreaded mode for debugging (and emergencies)
	if GLOBAL.singleThreadedBool {
		GLOBAL.numberOfCores = 1
	} else {
		GLOBAL.numberOfCores = runtime.NumCPU()
	}
	logInfo(fName, fmt.Sprintf("using %v threads", getNumberString(GLOBAL.numberOfCores)))

	// debugMultipler is used to slow down communication processes for easier observation
	if GLOBAL.debugModeBool {
		GLOBAL.debugMultiplier = 10
	} else {
		GLOBAL.debugMultiplier = 1
	}

	if GLOBAL.enableBackendBool {
		GLOBAL.backendData.Lock()
		GLOBAL.backendData.GalaxyName = GLOBAL.galaxyName
		GLOBAL.backendData.Unlock()
	}

}
