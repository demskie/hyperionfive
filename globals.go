package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/AlexDemskie/simplesync"
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
	PendingPlayerSpawns []*playerInfo
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// GalaxyStruct ...
type GalaxyStruct struct {
	// holds protobuf prestructured data
	Protobuf *Galaxy

	// created when generating galaxy to ensure objects are far enough apart
	Rtree *rtreego.Rtree

	// secondary random number generator for use outside ProcessingQueue
	SingleRnumMutex *sync.Mutex
	SingleRnum      *rand.Rand

	// implement variable tick rate by multiplying against TimeMultiplier
	TickTime       *tickCounterData
	TimeMultiplier float64

	// worker queue implementation
	ProcessingQueue     *simplesync.WorkerPool
	ProcessingRnumSlice []*rand.Rand

	// all objects that have a population
	PopulationObjectsMutex *sync.Mutex
	PopulationObjectsSlice []*SolarObject

	// used when spawning new players
	NeutralTerranPlanetMutex *sync.Mutex
	NeutralTerranPlanetSlice []*SolarObject

	// used to get pointers based on ID or Name
	PointerMappings *pointerMappings

	// used when serializing data for players
	PlayerFogOfWarMutex *sync.Mutex
	PlayerFogOfWar      map[string]map[string]*SolarSystem // playerName // sysName // sysPtr
}

// Initialize preallocates the variables
func (g *GalaxyStruct) Initialize() {
	g.Protobuf = &Galaxy{}
	g.Rtree = &rtreego.Rtree{}
	g.SingleRnumMutex = &sync.Mutex{}
	g.SingleRnum = rand.New(rand.NewSource(time.Now().UnixNano() - 99))
	g.TickTime = &tickCounterData{lastTick: time.Now()}
	g.TimeMultiplier = float64(1)
	g.ProcessingQueue = simplesync.NewWorkerPool(GLOBAL.numberOfCores)
	g.ProcessingRnumSlice = make([]*rand.Rand, GLOBAL.numberOfCores)
	for i := int64(0); i < int64(GLOBAL.numberOfCores); i++ {
		g.ProcessingRnumSlice[i] = rand.New(rand.NewSource(i + time.Now().UnixNano()))
	}
	g.PopulationObjectsMutex = &sync.Mutex{}
	g.PopulationObjectsSlice = make([]*SolarObject, 0, oneMillion)
	g.NeutralTerranPlanetMutex = &sync.Mutex{}
	g.NeutralTerranPlanetSlice = make([]*SolarObject, 0, oneMillion)
	g.PointerMappings = &pointerMappings{}
	g.PointerMappings.Initialize()
	g.PlayerFogOfWarMutex = &sync.Mutex{}
	g.PlayerFogOfWar = make(map[string]map[string]*SolarSystem, maxNumberOfPlayers)
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

// IsPlanet ...
func (obj *SolarObject) IsPlanet() bool {
	if obj.Type < 10 {
		return true
	}
	return false
}

// IsShip ...
func (obj *SolarObject) IsShip() bool {
	if (13 < obj.Type) && (obj.Type < 20) {
		return true
	}
	return false
}

// defining types of objects
const (
	TERRAN         int32 = iota // -20% population
	DESERT                      // -50% population, 2x logistics
	VOLCANIC                    // -75% population, lots of metal
	ICE                         // -50% population, lots of crystal
	BARREN                      // -90% population, 3x logistics
	FERROUS                     //
	GREENHOUSE                  //
	OCEANIC                     // max population
	LARGEASTEROID               //
	DWARF                       //
	_                           // _10_
	ASTEROIDBELT                //
	PLASMASTORM                 //
	_                           // _13_
	LARGETRADESHIP              // spawns and moves automatically from trading nations
	FIGHTER                     // close range combat
	FLAKCORVETTE                // block point-defense damage
	COLONIZER                   // colonize a single planet in new system
	CAPITALSHIP                 // long range ship-to-ship combat
	TITAN                       // can colonize, block point-defense and engage long-range
	_                           // _20_
)

// solarsystem type
const (
	YELLOWGIANT int32 = iota // most desirable
	REDGIANT                 // mostly lava
	BLUEGIANT                // mostly ice
	NEUTRONSTAR              // just gas clouds
	WHITEDWARF               // just asteroids
	BLACKHOLE                // random, dangerous and AI spawn here
)

// galaxy type
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

// empty string on obj.Type() means NEUTRAL
const (
	NEUTRAL string = ""
)

// used to determine what the odds of success will be
const (
	ATTACKING int32 = iota
	RUSHING
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
	GLOBAL.clientBlacklist.LiveConnections = map[string][]string{}
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
