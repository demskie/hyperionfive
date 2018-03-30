package main

import (
	"crypto/tls"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"time"
	// runtimepprof "runtime/pprof"
)

func startBackendMode() {
	const fName = "(backend)"

	// this type and method is defined in globals.go
	galaxy := &GalaxyStruct{}
	galaxy.Initialize()

	// create or deserialize a protobuf galaxy
	if GLOBAL.inputFile == "" {
		galaxy.Generate()
	} else {
		galaxy.UnmarshalFile(GLOBAL.inputFile)
	}

	// preprocess some large datastructures
	galaxy.PreprocessPointerMappings()
	galaxy.PreprocessPopulationObjects()
	galaxy.PreprocessPlayerFogOfWar()

	// connect to a frontend server
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
				&playerInfo{
					uniqueID: "AlexDemskie",
					strength: 1,
				})
			GLOBAL.galaxyInfo.Unlock()
		}
		// start monitoring backend atomic counters
		backendCounters.StartCounting()
		// start processing loop
		for {
			galaxy.TimeMultiplier = galaxy.TickTime.Update()

			GLOBAL.galaxyInfo.Lock()
			galaxy.SpawnNewPlayers()
			galaxy.SpawnFighters()
			galaxy.CalculatePopulation()
			galaxy.IncreaseFoodOnNeutralPlanets()
			galaxy.ProcessOngoingBattles()
			GLOBAL.galaxyInfo.Unlock()

			close(GLOBAL.clientSignal)
			GLOBAL.clientSignal = make(chan struct{})
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

	// wait for program to signal exit
	<-GLOBAL.exitProgram
	return
}
