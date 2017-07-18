package main

import (
	"container/ring"
	"math"
	"sync"
	"time"
)

// GLOBAL declaration allows easy access to these truly global variables
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

	turningDataSignal chan struct{}
	newTickSignal     chan struct{}
	serverPriority    uint64
	numberOfCores     int
	loggingMutex      sync.Mutex
	waitForLogging    sync.WaitGroup
	waitForCores      map[string]*sync.WaitGroup

	clientBlacklist clientBlacklist

	currentGalaxy           Galaxy
	currentSystemHashes     map[string][]byte
	futureGalaxy            Galaxy
	partitionedSolarSystems []sync.RWMutex
	parsedPlayerPtrs        map[string][]*SolarSystem
	uniqueRequestID         chan int64

	bufferedRequests       []incomingClientRequest
	bufferedRequestsMutex  sync.RWMutex
	confirmedRequests      map[string][]int64
	confirmedRequestsMutex sync.RWMutex

	backendData backendData

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
	Frontends                 []string
	ClientIDs                 map[string][]string
	BackendSystemHashes       map[string]map[string][]byte
	BackendAdvertisedPriority map[string]uint64
	BackendLatency            map[string]*ring.Ring
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
	Frontends           []string
	Backends            []string
	Priority            uint64
	CurrentSystemHashes map[string][]byte
	BufferedRequests    []incomingClientRequest
	ConfirmedRequests   []int64
	SystemData          []SolarSystem
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
	ServerAddress     string   `json:"ServerAddress"`
	GalaxyName        string   `json:"GalaxyName"`
	ActiveClientCount int      `json:"ActiveClientCount"`
	TotalClientCount  int      `json:"TotalClientCount"`
	OtherFrontends    []string `json:"OtherFrontends"`
	OtherBackends     []string `json:"OtherBackends"`
	Message           string   `json:"Message"`
	AnotherMessage    string   `json:"AnotherMessage"`
}

type frontendToBackendMessage struct {
	GalaxyName     string   `json:"GalaxyName"`
	GlobalX        int64    `json:"GlobalX"`
	GlobalY        int64    `json:"GlobalY"`
	OtherFrontends []string `json:"OtherFrontends"`
	OtherBackends  []string `json:"OtherBackends"`
	Message        string   `json:"Message"`
	AnotherMessage string   `json:"AnotherMessage"`
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
