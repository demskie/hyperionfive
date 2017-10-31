package main

import (
	"container/ring"
	"math"
	"sync"
	"time"
)

// GLOBAL declaration allows easy access to these global variables
var GLOBAL globalVars

type globalVars struct {
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

	logInfoChan chan string

	serverPriority uint64
	numberOfCores  int
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
	waitForCores   map[string]*sync.WaitGroup

	clientBlacklist clientBlacklist

	turningDataSignal     chan struct{}
	newTickSignal         chan struct{}
	mutatingDataBlocker   chan struct{}
	allServersReadySignal chan struct{}
	currentGalaxy         Galaxy
	futureGalaxy          Galaxy
	uniqueRequestID       chan int64
	bufferedRequests      []incomingClientRequest
	bufferedRequestsMutex sync.RWMutex
	backendData           backendData

	frontendData frontendData
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type clientBlacklist struct {
	sync.RWMutex
	TotalSessionCount       map[string]uint16
	FailedChallengeCount    map[string]uint16
	FailedChallengeTime     map[string]int64
	FailedGoogleAuthCount   map[string]uint16
	FailedGoogleAuthTime    map[string]int64
	FailedSessionLimitCount map[string]uint16
	FailedSessionLimitTime  map[string]int64
}

type incomingClientRequest struct {
	ClientID      string   `json:"clientID"`
	UniqueID      int64    `json:"uniqueID"`
	ClientMessage string   `json:"clientMessage"`
	ClientValues  []string `json:"clientValues"`
}

type backendData struct {
	sync.RWMutex
	Frontends           []string
	Backends            []string
	ClientIDs           map[string][]string
	SystemBytes         map[string][]byte
	SystemHashes        [16]map[string]map[string][4]byte
	ParsedPlayerPtrs    map[string][]*SolarSystem
	Priorities          map[string]uint64
	Latencies           map[string]*ring.Ring
	TickTime            int64
	NeighborTickTimes   map[string]uint64
	ReceivedMessageFrom []string
	ConfirmedRequestIDs map[string][]int64
}

type frontendData struct {
	sync.RWMutex
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalGalaxyType  map[string]int32
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type backendToBackendMessage struct {
	Frontends         []string
	Backends          []string
	Priority          uint64
	SystemHashes      [16]map[string][4]byte
	BufferedRequests  []incomingClientRequest
	AckOnly           bool
	ConfirmedRequests []int64
	SystemBytes       map[string][]byte
}

type frontendToFrontendMessage struct {
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalGalaxyType  map[string]int32
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends []string
	Message        string
	AnotherMessage string
}

type backendToFrontendMessage struct {
	ServerAddress     string
	GalaxyName        string
	ActiveClientCount int
	OtherFrontends    []string
	OtherBackends     []string
	Message           string
	AnotherMessage    string
}

type frontendToBackendMessage struct {
	GalaxyName     string
	GlobalX        int64
	GlobalY        int64
	OtherFrontends []string
	OtherBackends  []string
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
	connectedBackendsLimit     = 3
	idleConnectedBackendsLimit = 25
	latencyRingLength          = 25
	backendTickrate            = 1 * time.Second
)

const (
	numDimensions   = 2
	minBranchFactor = 2
	maxBranchFactor = 2
)

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
