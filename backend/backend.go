package backend

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/demskie/hyperionbot/actor"
	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/hyperionfive/names"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/demskie/hyperionfive/stringdatastore"
	"github.com/demskie/logging"
	"github.com/demskie/randutil"
	"github.com/demskie/spatialindex"

	"github.com/demskie/simplesync"
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
		Protobuf:                 &gamestate.Galaxy{},
		Rtree:                    spatialindex.NewTree(),
		SingleRnumMutex:          &sync.Mutex{},
		SingleRnum:               randutil.CreateUniqueMathRnum(),
		TickTime:                 &tickCounterData{lastTick: time.Now()},
		TimeMultiplier:           float64(1),
		ProcessingQueue:          simplesync.NewWorkerPool(args.NumberOfCores),
		ProcessingRnumSlice:      randutil.CreateMultipleUniqueMathRnum(args.NumberOfCores),
		AnyPlanetMutex:           &sync.Mutex{},
		AnyPlanetSlice:           make([]*gamestate.Planet, 0),
		NeutralAnyPlanetMutex:    &sync.Mutex{},
		NeutralAnyPlanetSlice:    make([]*gamestate.Planet, 0),
		NeutralTerranPlanetMutex: &sync.Mutex{},
		NeutralTerranPlanetSlice: make([]*gamestate.Planet, 0),
		PointerMappings:          NewPointerMappings(),
		PlayerFogOfWarMutex:      &sync.Mutex{},
		PlayerFogOfWar:           make(map[uint64]map[uint64]*gamestate.System, maxNumberOfPlayers),
		SystemAndPlanetNames:     names.NewNameGenerator(args, log),
		FleetIDs:                 intutil.NewInt32Generator(),
		BattleIDs:                intutil.NewInt32Generator(),
		SiegeIDs:                 intutil.NewInt32Generator(),
		ActionIDs:                intutil.NewInt32Generator(),
		SystemNameLookup:         make(map[uint64]*gamestate.System),
		PlayerFleetInsideSystem:  make(map[uint64]map[uint64]*gamestate.Fleet),
	}

	universe := &UniverseStruct{
		GalaxyNameUint:       uint64(math.MaxUint64),
		GlobalX:              int64(0),
		GlobalY:              int64(0),
		ClientSignal:         make(chan struct{}),
		PlayerPtrs:           make(map[uint64][]*gamestate.System),
		PlayerNameData:       stringdatastore.New(),
		SystemAndPlanetNames: galaxy.SystemAndPlanetNames,
		PendingPlayerSpawns:  make([]*playerInfo, 0),
		Firewall:             nil,
		Actors:               make([]*actor.Bot, 0),
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
	galaxy.PreprocessAllocations()

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
					nameUint: 123456789,
					strength: 1,
				})

			universe.Unlock()
		}
		// start monitoring backend atomic counters
		backendCounters.StartCounting()
		// start processing loop
		for {
			galaxy.TimeMultiplier = galaxy.TickTime.Update()

			if galaxy.TimeMultiplier >= 1 {
				log.Error(fName, fmt.Sprintf("iteration took %v seconds", galaxy.TimeMultiplier))
			}

			universe.Lock()
			galaxy.SpawnNewPlayers(universe) // processing_playerspawn.go
			galaxy.CalculatePopulation()     // processing_population.go
			galaxy.AdvanceProduction()       // processing_production.go
			galaxy.ProcessOngoingBattles()   // processing_battle.go
			galaxy.MoveFleets()              // processing_movefleets.go

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
	realMilkyWayRadius            = 100000
	lightYear                     = galaxyRadius / realMilkyWayRadius
	minimumDistanceBetweenSystems = 3 * lightYear

	oneMillion = 1000000
	oneBillion = 1000000000
)

// used to determine what the odds of success will be
const (
	ATTACKING int32 = iota
	RUSHING
)
