package backend

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/demskie/hyperionfive/auth"

	"github.com/alecthomas/units"
	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/backendmessage"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/demskie/logging"
	"github.com/demskie/randutil"
	"github.com/demskie/spatialindex"
	"github.com/gorilla/websocket"

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
		Protobuf:                 &backendmessage.Galaxy{},
		Rtree:                    spatialindex.NewTree(),
		SingleRnum:               randutil.CreateUniqueMathRnum(),
		TickTime:                 &tickCounterData{lastTick: time.Now()},
		TimeMultiplier:           float64(1),
		AnyPlanetSlice:           make([]*backendmessage.Planet, 0),
		NeutralAnyPlanetSlice:    make([]*backendmessage.Planet, 0),
		NeutralTerranPlanetSlice: make([]*backendmessage.Planet, 0),
		FleetIDs:                 intutil.NewInt32Generator(),
		BattleIDs:                intutil.NewInt32Generator(),
		SiegeIDs:                 intutil.NewInt32Generator(),
		ActionIDs:                intutil.NewInt32Generator(),
	}

	universe := &UniverseStruct{
		GalaxyStruct:        galaxy,
		UgridSystems:        spatialindex.NewGrid(65535),
		UgridFleets:         spatialindex.NewGrid(65535),
		UgridBattles:        spatialindex.NewGrid(65535),
		UgridSieges:         spatialindex.NewGrid(65535),
		UgridBytes:          map[int]bytes.Buffer{},
		ClientSignal:        make(chan struct{}),
		PendingPlayerSpawns: make([]*playerInfo, 0),
	}

	// create or deserialize a protobuf galaxy
	if args.InputFile == "" {
		galaxy.Generate()
	} else {
		galaxy.UnmarshalFile()
	}

	// create uGrid for virtualized client systems
	for systemName, system := range universe.GalaxyStruct.Protobuf.System {
		universe.UgridSystems.Add(systemName, system.InterX, system.InterY)
	}

	// preprocess some large datastructures
	galaxy.PreprocessPopulationObjects()

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
				},
			)
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
			galaxy.AdvanceActors()           // processing_actors.go
			galaxy.ProcessActions()          // processing_actions.go
			galaxy.CalculatePopulation()     // processing_population.go
			galaxy.AdvanceProduction()       // processing_production.go
			galaxy.ProcessOngoingBattles()   // processing_battle.go
			universe.Unlock()

			close(universe.ClientSignal)
			universe.ClientSignal = make(chan struct{})
		}
	}()

	////////////////////////////////////////////////////////////
	// setup HTTP and HTTPS server
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()
	muxSecure.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		err := fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
		if log.CheckError(err, fName, httpRemoteIP) {
			return
		}
		upgrader := websocket.Upgrader{
			HandshakeTimeout:  15 * time.Second,
			ReadBufferSize:    0,
			WriteBufferSize:   0,
			EnableCompression: true,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if log.CheckError(err, fName, "upgrade error") {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(128 * int64(units.Kilobyte))
		switch conn.Subprotocol() {
		case "detailedData":
			userID, err := auth.HandleGoogleAuth(fName, conn, nil, fwall, args, log)
			if log.CheckError(err, fName) {
				return
			}
			conn.SetReadLimit(1 * int64(units.Megabyte))
			universe.ClientToBackendCommunication(conn, userID)
		default:
			log.Info(fName, "received invalid Subprotocol", conn.Subprotocol())
		}
	})

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

		// allow HTTP2 by not specifying proto handlers
		TLSNextProto: nil,
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

	oneMillion = 1000000
	oneBillion = 1000000000
)

// used to determine what the odds of success will be
const (
	ATTACKING int32 = iota
	RUSHING
)
