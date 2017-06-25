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
	newTickSignal     chan struct{}
	serverPriority    uint64
	numberOfCores     int
	loggingMutex      sync.Mutex
	waitForLogging    sync.WaitGroup
	waitForCores      map[string]*sync.WaitGroup

	clientBlacklist         map[string]*firewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex

	currentGalaxy           Galaxy
	currentGalaxyHash       []byte
	futureGalaxy            Galaxy
	partitionedSolarSystems []sync.RWMutex
	parsedPlayerPtrs        map[string][]*SolarSystem

	getUniqueRequestID     chan int64
	bufferedRequests       []incomingRequest
	bufferedRequestsMutex  sync.RWMutex
	confirmedRequests      map[string][]int64
	confirmedRequestsMutex sync.RWMutex

	backendData           backendDataStruct
	backendDataMutex      sync.RWMutex
	fullBackendStats      []backendInfoStruct
	fullBackendStatsMutex sync.RWMutex

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
	ClientID      string   `json:"clientID"`
	UniqueID      int64    `json:"uniqueID"`
	ClientMessage string   `json:"clientMessage"`
	ClientValues  []string `json:"clientValues"`
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
	ListOfGalaxies []galInfo `json:"listOfGalaxies"`
	OtherFrontends []string  `json:"otherFrontends"`
	Message        string    `json:"message"`
	AnotherMessage string    `json:"anotherMessage"`
}
type galInfo struct {
	GalaxyName        string   `json:"galaxyName"`
	GlobalX           int64    `json:"globalX"`
	GlobalY           int64    `json:"globalY"`
	ClientIDs         []string `json:"clientIDs"`
	ClusterOfBackends []string `json:"clusterOfBackends"`
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
