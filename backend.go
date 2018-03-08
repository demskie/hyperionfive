package main

import (
	"crypto/tls"
	"errors"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dhconnelly/rtreego"
	// runtimepprof "runtime/pprof"
)

func startBackendMode() {
	const fName = "(backend)"

	////////////////////////////////////////////////////////////
	// create or recover galaxy
	////////////////////////////////////////////////////////////

	galaxy := &GalaxyData{
		Data:              &Galaxy{},
		TickTime:          &tickCounterData{lastTick: time.Now()},
		ProcessingQueue:   make(chan func(coreNum int), 1),
		TimeMultiplier:    float64(1),
		RnumSlice:         make([]*rand.Rand, GLOBAL.numberOfCores),
		TitanPtrs:         make([]*SolarObject, 0, oneMillion),
		PlanetPtrs:        make([]*SolarObject, 0, oneMillion),
		TerranPlanetPtrs:  make([]*SolarObject, 0, oneMillion),
		NeutralPlanetPtrs: make([]*SolarObject, 0, oneMillion),
		Rtree:             &rtreego.Rtree{},
		PlayerPtrs:        make(map[string][]*SolarSystem, oneMillion),
		ObjectNamePtrs:    make(map[string]*SolarObject, oneMillion),
		SystemNamePtrs:    make(map[string]*SolarSystem, oneMillion),
		GalacticFleetPtrs: make(map[string]*GalacticFleet, oneMillion),
		SystemLinkPtrs:    make(map[string][]*SolarSystem, oneMillion),
	}

	// create a random number generator for each core
	seed := time.Now().UnixNano()
	for coreID := range galaxy.RnumSlice {
		seed += int64(coreID)
		galaxy.RnumSlice[coreID] = rand.New(rand.NewSource(seed))
	}

	// create or deserialize a galaxy
	if GLOBAL.inputFile == "" {
		galaxy.Generate()
	} else {
		galaxy.UnmarshalFile(GLOBAL.inputFile)
	}

	////////////////////////////////////////////////////////////
	// initialize worker pool
	////////////////////////////////////////////////////////////

	workerWaitGroup := sync.WaitGroup{}
	workFuncPerCore := make(chan func(coreNum int), 1)
	go func() {
		for {
			workerWaitGroup.Add(GLOBAL.numberOfCores)
			workFunc := <-galaxy.ProcessingQueue
			for i := 0; i < GLOBAL.numberOfCores; i++ {
				workFuncPerCore <- workFunc
			}
			workerWaitGroup.Wait()
		}
	}()
	for cn := 0; cn < GLOBAL.numberOfCores; cn++ {
		go func(coreNum int) {
			for {
				workToDo := <-workFuncPerCore
				workToDo(coreNum)
				workerWaitGroup.Done()
			}
		}(cn)
	}

	////////////////////////////////////////////////////////////
	// preprocess some large datastructures
	////////////////////////////////////////////////////////////

	galaxy.GatherAllPlayerPointers()
	galaxy.GatherAllPlanetPointers()
	//galaxy.GatherAllTitanPointers()
	galaxy.GatherAllObjectAndSystemNames()

	backendCounters.StartCounting()

	////////////////////////////////////////////////////////////
	// connect to a frontend server
	////////////////////////////////////////////////////////////

	if false {
		go func() {
			for {
				spawnBackendToFrontend("hyperion5.com", galaxy)
				time.Sleep(1 * time.Second)
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// continuously process galaxy
	////////////////////////////////////////////////////////////

	go func() {

		// for testing purposes
		if GLOBAL.debugModeBool {
			GLOBAL.galaxyInfo.Lock()
			GLOBAL.galaxyInfo.PendingPlayerSpawns = append(
				GLOBAL.galaxyInfo.PendingPlayerSpawns,
				playerInfo{
					uniqueID: "AlexDemskie",
					strength: 1,
				})
			GLOBAL.galaxyInfo.Unlock()
		}

		// start processing loop
		for {
			galaxy.TimeMultiplier = galaxy.TickTime.Update()

			GLOBAL.galaxyInfo.Lock()
			galaxy.SpawnNewPlayers()
			galaxy.CalculatePopulation()
			galaxy.IncreaseFoodOnNeutralPlanets()
			galaxy.ProcessOngoingBattles()
			GLOBAL.galaxyInfo.Unlock()

			close(GLOBAL.clientSignal)
			GLOBAL.clientSignal = make(chan struct{})

			galaxy.TickTime.WaitOneMillisecondAndReset()
		}

	}()

	////////////////////////////////////////////////////////////
	// setup HTTP and HTTPS server
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	// setup the websocket handlers
	muxSecure.HandleFunc("/clienttobackend", handleClientToBackend)

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         getLocalAddress().IP.String() + ":443",
		Handler:      muxSecure,

		// defining various security parameters used to specify accepted ciphersuites
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// serve HTTPS requests with config defined above
	go func() {
		exPath, err := os.Executable()
		if checkError(err, fName) {
			suddenExit()
		}
		dir := filepath.Dir(exPath)
		errServeTLS := srvSecure.ListenAndServeTLS(
			filepath.Join(dir, "server.rsa.crt"),
			filepath.Join(dir, "server.rsa.key"))
		if checkError(errServeTLS, fName) {
			suddenExit()
		}
	}()

	// serve HTTP requests just for Debug purposes
	if GLOBAL.debugModeBool {
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
				Addr:    getLocalAddress().IP.String() + ":8080",
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			checkError(errServeInsecure, fName, "(net/http/pprof)")
		}()
	}

	if GLOBAL.debugModeBool {
		// f, err := os.Create("cpuprofile.blob")
		// checkError(err, fName, "(runtime/pprof)")
		// runtimepprof.StartCPUProfile(f)
		// defer runtimepprof.StopCPUProfile()
	}

	// wait for program to signal exit
	<-GLOBAL.exitProgram
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

type weightedObj struct {
	Weight int
	Item   interface{}
}

func weightedChoice(choices []weightedObj, rnum *rand.Rand) (weightedObj, error) {
	max := 0
	for _, choice := range choices {
		max += choice.Weight
	}
	num := rnum.Intn(max)
	//logDebug(fmt.Sprint("num ", num))
	for _, choice := range choices {
		num -= choice.Weight
		if num <= 0 {
			//logDebug(fmt.Sprint("returning ", choice))
			return choice, nil
		}
		//logDebug(fmt.Sprint("skipping ", choice))
	}
	err := errors.New("weightedChoice func is broken")
	return weightedObj{}, err
}
