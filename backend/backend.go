package backend

import (
	"crypto/tls"
	"hyperionfive/arguments"
	"hyperionfive/firewall"
	"hyperionfive/logging"
	"hyperionfive/names"
	"hyperionfive/netutil"
	"hyperionfive/randutil"
	"hyperionfive/simpletree"
	"hyperionfive/stringdatastore"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AlexDemskie/simplesync"
	// runtimepprof "runtime/pprof"
	_ "net/http/pprof"
)

var (
	args  *arguments.Args
	log   *logging.Logger
	fwall *firewall.Checker
)

// Start ...
func Start(arguments *arguments.Args, logger *logging.Logger) {
	const fName = "(backend)"

	args = arguments
	log = logger
	fwall = firewall.New()

	galaxy := &GalaxyStruct{
		Protobuf:                 &Galaxy{},
		Rtree:                    simpletree.NewTree(),
		SingleRnumMutex:          &sync.Mutex{},
		SingleRnum:               randutil.GetUniqueMathRnum(),
		TickTime:                 &tickCounterData{lastTick: time.Now()},
		TimeMultiplier:           float64(1),
		ActorStateSlice:          make([]*ActorState, 0),
		ProcessingQueue:          simplesync.NewWorkerPool(args.NumberOfCores),
		ProcessingRnumSlice:      randutil.GetMultipleUniqueMathRnum(args.NumberOfCores),
		AnyPlanetMutex:           &sync.Mutex{},
		AnyPlanetSlice:           make([]*SolarObject, 0),
		NeutralAnyPlanetMutex:    &sync.Mutex{},
		NeutralAnyPlanetSlice:    make([]*SolarObject, 0),
		NeutralTerranPlanetMutex: &sync.Mutex{},
		NeutralTerranPlanetSlice: make([]*SolarObject, 0),
		PointerMappings:          NewPointerMappings(),
		PlayerFogOfWarMutex:      &sync.Mutex{},
		PlayerFogOfWar:           make(map[int32]map[uint64]*SolarSystem, maxNumberOfPlayers),
		SystemAndPlanetNames:     names.NewNameGenerator(args, log),
	}

	universe := &UniverseStruct{
		GalaxyNameInt:        int32(-1),
		GlobalX:              int64(0),
		GlobalY:              int64(0),
		ClientSignal:         make(chan struct{}),
		PlayerPtrs:           make(map[int32][]*SolarSystem),
		PlayerNameData:       stringdatastore.New(),
		SystemAndPlanetNames: galaxy.SystemAndPlanetNames,
		PendingPlayerSpawns:  make([]*playerInfo, 0),
	}

	// create or deserialize a protobuf galaxy
	if args.InputFile == "" {
		galaxy.Generate()
	} else {
		galaxy.UnmarshalFile()
	}

	// preprocess some large datastructures
	galaxy.PreprocessPointerMappings()
	galaxy.PreprocessPopulationObjects()
	galaxy.PreprocessPlayerFogOfWar()

	// connect to a frontend server
	if false {
		go func() {
			for {
				universe.SpawnBackendToFrontend("hyperion5.com", galaxy)
				time.Sleep(1 * time.Second)
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// continuously process galaxy
	////////////////////////////////////////////////////////////

	go func() {
		// for testing purposes
		if args.DebugModeBool {
			universe.Lock()
			universe.PendingPlayerSpawns = append(
				universe.PendingPlayerSpawns,
				&playerInfo{
					uniqueID: "1234567890987654321",
					strength: 1,
				})
			universe.Unlock()
		}
		// start monitoring backend atomic counters
		backendCounters.StartCounting()
		// start processing loop
		for {
			galaxy.TimeMultiplier = galaxy.TickTime.Update()

			universe.Lock()
			galaxy.SpawnNewPlayers(universe)      // processing_playerspawn.go
			galaxy.SpawnFighters()                // processing_population.go
			galaxy.CalculatePopulation()          // processing_population.go
			galaxy.IncreaseFoodOnNeutralPlanets() // processing_population.go
			galaxy.ProcessOngoingBattles()        // processing_battle.go
			galaxy.ProcessActors()                // processing_actor.go
			universe.Unlock()

			close(universe.ClientSignal)
			universe.ClientSignal = make(chan struct{})
		}
	}()

	////////////////////////////////////////////////////////////
	// setup HTTP and HTTPS server
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	// setup the websocket handlers
	muxSecure.HandleFunc("/clienttobackend", universe.HandleClientToBackend)

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         netutil.GetLocalAddress(args, log) + ":443",
		Handler:      muxSecure,

		// defining various security parameters used to specify accepted ciphersuites
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				//tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				//tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				//tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				//tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// serve HTTPS requests with config defined above
	go func() {
		exPath, err := os.Executable()
		if log.CheckError(err, fName) {
			log.SuddenExit()
		}
		dir := filepath.Dir(exPath)
		errServeTLS := srvSecure.ListenAndServeTLS(
			filepath.Join(dir, "server.rsa.crt"),
			filepath.Join(dir, "server.rsa.key"))
		if log.CheckError(errServeTLS, fName) {
			log.SuddenExit()
		}
	}()

	// serve HTTP requests just for Debug purposes
	if args.DebugModeBool {
		go func() {
			profmux := http.NewServeMux()
			profmux.HandleFunc("/debug/pprof/", pprof.Index)
			profmux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			profmux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			profmux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			profmux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			profmux.Handle("/debug/pprof/block", pprof.Handler("block"))
			profmux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			profmux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			profmux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			profmux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
			srvProf := &http.Server{
				Addr:    netutil.GetLocalAddress(args, log) + ":8080",
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			log.CheckError(errServeInsecure, fName, "(net/http/pprof)")
		}()
	}

	indefinite := make(chan struct{})
	<-indefinite

	return
}

const (
	maxNumberOfPlayers  = 10000
	numSystemsPerPlayer = 25
	totalNumSystems     = maxNumberOfPlayers * numSystemsPerPlayer

	galaxyRadius                  = math.MaxInt64
	realLifeGalaxyRadius          = 125000
	lightyear                     = galaxyRadius / realLifeGalaxyRadius
	minimumDistanceBetweenSystems = 7 * lightyear

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

// used to determine what the odds of success will be
const (
	ATTACKING int32 = iota
	RUSHING
)
