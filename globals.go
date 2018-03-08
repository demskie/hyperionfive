package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/dhconnelly/rtreego"
)

// GLOBAL variables
var GLOBAL globalVars

type globalVars struct {
	inputFile          string
	outputFile         string
	googleAudience     string
	authPassword       string
	staticFrontend     string
	localInterface     string
	localAddress       string
	enableFrontendBool bool
	enableBackendBool  bool
	debugModeBool      bool
	singleThreadedBool bool

	exitProgram chan struct{}

	logInfoChan chan loggingMessage

	numberOfCores int

	clientSignal      chan struct{}
	clientBlacklist   clientBlacklist
	clientRequests    clientRequests
	clientConnections clientConnections

	galaxyInfo galaxyInfo
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

type clientRequests struct {
	sync.Mutex
	Data []modifiedClientRequest
}

type modifiedClientRequest struct {
	ClientID      string
	ClientMessage string
	ClientValues  []string
}

type clientConnections struct {
	sync.Mutex
	Data map[string]time.Time
}

type galaxyInfo struct {
	sync.RWMutex
	GalaxyName          string
	GlobalX             int64
	GlobalY             int64
	PlayerPtrs          map[string][]*SolarSystem
	PendingPlayerSpawns []playerInfo
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// GalaxyData ...
type GalaxyData struct {
	Data              *Galaxy
	TickTime          *tickCounterData
	ProcessingQueue   chan func(coreNum int)
	TimeMultiplier    float64
	RnumSlice         []*rand.Rand
	PlanetPtrs        []*SolarObject
	TerranPlanetPtrs  []*SolarObject
	NeutralPlanetPtrs []*SolarObject
	TitanPtrs         []*SolarObject
	Rtree             *rtreego.Rtree
	PlayerPtrs        map[string][]*SolarSystem
	ObjectNamePtrs    map[string]*SolarObject
	SystemNamePtrs    map[string]*SolarSystem
	GalacticFleetPtrs map[string]*GalacticFleet
	SystemLinkPtrs    map[string][]*SolarSystem
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type frontendToFrontendMessage struct {
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends map[string]time.Time
}

type backendToFrontendMessage struct {
	GalaxyName string
	GlobalX    int64
	GlobalY    int64
	ClientIDs  []string
}

type playerInfo struct {
	uniqueID string
	strength int
}

type frontendToBackendMessage struct {
	GalaxyName     string
	GlobalX        int64
	GlobalY        int64
	PlayersToSpawn []playerInfo
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
	maxNumberOfPlayers  = 10000
	numSystemsPerPlayer = 10
	totalNumSystems     = maxNumberOfPlayers * numSystemsPerPlayer

	galaxyRadius                  = math.MaxInt64
	realLifeGalaxyRadius          = 125000
	fakeToRealMultiplier          = galaxyRadius / realLifeGalaxyRadius
	lightyear                     = fakeToRealMultiplier
	minimumDistanceBetweenSystems = lightyear

	oneMillion  = 1000000
	oneBillion  = 1000000000
	oneTrillion = 1000000000000
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

	GLOBAL.numberOfCores = 0

	GLOBAL.clientSignal = make(chan struct{})
	GLOBAL.clientBlacklist = clientBlacklist{}
	GLOBAL.clientBlacklist.LiveConnections = map[string][]string{} // remoteIP : fName
	GLOBAL.clientBlacklist.FailedChallengeCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedGoogleAuthCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedGoogleAuthTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedSessionLimitCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedSessionLimitTime = map[string]time.Time{}
	GLOBAL.clientRequests = clientRequests{}
	GLOBAL.clientRequests.Data = make([]modifiedClientRequest, 0, oneMillion)
	GLOBAL.clientConnections = clientConnections{}
	GLOBAL.clientConnections.Data = map[string]time.Time{}

	GLOBAL.galaxyInfo = galaxyInfo{}
	GLOBAL.galaxyInfo.GalaxyName = "UNASSIGNED"
	GLOBAL.galaxyInfo.GlobalX = int64(123456789)
	GLOBAL.galaxyInfo.GlobalY = int64(123456789)
	GLOBAL.galaxyInfo.PlayerPtrs = map[string][]*SolarSystem{}

	// define commandline flags to parse
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "override export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "override google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	staticFrontendPtr := flag.String("staticfrontend", "", "specify static frontend address")
	localInterfacePtr := flag.String("interface", "", "override local interface to use")
	localAddressPtr := flag.String("localip", "", "override local ip to use")
	enableFrontendBoolPtr := flag.Bool("frontend", false, "enable frontend mode")
	enableBackendBoolPtr := flag.Bool("backend", false, "enable backend mode")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")
	singleThreadedBoolPtr := flag.Bool("singlethreaded", false, "force singlethreaded behavior")

	// parse the arguments
	flag.Parse()
	GLOBAL.inputFile = *inputFilePtr
	GLOBAL.outputFile = *outputFilePtr
	GLOBAL.googleAudience = *googleAudiencePtr
	GLOBAL.authPassword = *authPasswordPtr
	GLOBAL.staticFrontend = *staticFrontendPtr
	GLOBAL.localInterface = *localInterfacePtr
	GLOBAL.localAddress = *localAddressPtr
	GLOBAL.enableFrontendBool = *enableFrontendBoolPtr
	GLOBAL.enableBackendBool = *enableBackendBoolPtr
	GLOBAL.debugModeBool = *debugModeBoolPtr
	GLOBAL.singleThreadedBool = *singleThreadedBoolPtr

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

}
