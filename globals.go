package main

import (
	"container/ring"
	"math"
	"sync"
	"time"
)

// GLOBAL var allows me to clearly label which objects are global
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
	processingSignal  chan struct{}
	numberOfCores     int
	loggingMutex      sync.Mutex
	waitForLogging    sync.WaitGroup
	waitForCores      map[string]*sync.WaitGroup

	clientBlacklist         map[string]*firewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex

	sequenceNumber          uint64
	currentGalaxy           Galaxy
	futureGalaxy            Galaxy
	partitionedSolarSystems []sync.RWMutex
	parsedPlayerPtrs        map[string][]*SolarSystem

	bufferedRequests      []incomingRequest
	bufferedRequestsMutex sync.RWMutex
	confirmedRequests     []incomingRequest

	backendData           backendDataStruct
	backendDataMutex      sync.RWMutex
	fullBackendStats      []backendInfoStruct
	fullBackendStatsMutex sync.RWMutex
	idleBackendStats      []backendInfoStruct
	idleBackendStatsMutex sync.RWMutex

	frontendData      frontendDataStruct
	frontendDataMutex sync.RWMutex
}

type firewallClientValues struct {
	sync.RWMutex
	FailedChallengeCount    uint16
	FailedChallengeTime     int64
	FailedGoogleAuthCount   uint16
	FailedGoogleAuthTime    int64
	FailedSessionLimitCount uint16
	FailedSessionLimitTime  int64
}

type incomingRequest struct {
	ClientID      string
	ClientMessage string
	ClientValues  []string
	ServerMessage string
}

type backendDataStruct struct {
	ClientIDs             []string
	Frontends             []string
	Backends              []string
	HighestPriority       uint64
	SecondHighestPriority uint64
	GlobalX               int64
	GlobalY               int64
}

type backendInfoStruct struct {
	ServerIP           string
	AdvertisedPriority uint64
	Latency            *ring.Ring
}

type frontendDataStruct struct {
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
