package main

import (
	"container/ring"
	"math"
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
	frontendBool       bool
	backendBool        bool
	createGalaxyBool   bool
	debugModeBool      bool
	singleThreadedBool bool
	testModeBool       bool

	logInfoChan chan loggingMessage // see logging.go

	serverPriority  uint64
	numberOfCores   int
	loggingMutex    sync.Mutex
	waitForCores    sync.WaitGroup
	clientBlacklist clientBlacklist
	exitProgram     chan struct{}

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
	LiveConnections         map[string][]string
	FailedChallengeCount    map[string]uint16
	FailedChallengeTime     map[string]time.Time
	FailedGoogleAuthCount   map[string]uint16
	FailedGoogleAuthTime    map[string]time.Time
	FailedSessionLimitCount map[string]uint16
	FailedSessionLimitTime  map[string]time.Time
}

type incomingClientRequest struct {
	ClientID      string   `json:"clientID"`
	UniqueID      int64    `json:"uniqueID"`
	ClientMessage string   `json:"clientMessage"`
	ClientValues  []string `json:"clientValues"`
}

type backendData struct {
	sync.RWMutex
	GalaxyName          string
	GlobalX             int64
	GlobalY             int64
	Frontends           []string
	Backends            []string
	ClientIDs           map[string][]string // sysName : clientIDs?
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
	maxNumberOfPlayers  = 1000
	numSystemsPerPlayer = 100
	//totalNumSystems               = maxNumberOfPlayers * numSystemsPerPlayer
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
