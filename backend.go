package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	cryptorand "crypto/rand"

	"github.com/dhconnelly/rtreego"
	"github.com/golang/protobuf/proto"
)

func startBackendMode() {

	// initialize primary data structures
	GLOBAL.backendData = backendDataStruct{}
	GLOBAL.currentGalaxy = Galaxy{}
	GLOBAL.futureGalaxy = Galaxy{}

	////////////////////////////////////////////////////////////
	// generate randomNumber to determine server priority
	////////////////////////////////////////////////////////////

	enabled := true
	if enabled {
		serverPriority = generateRandomUint64()
		logInfo(fmt.Sprintf("serverPriority = %d", serverPriority))
	}

	////////////////////////////////////////////////////////////
	// handle CLI parameters
	////////////////////////////////////////////////////////////

	enabled = false
	if enabled {
		handleInputParameters("startBackendMode")
	}

	////////////////////////////////////////////////////////////
	// start communication with Frontend
	////////////////////////////////////////////////////////////

	enabled = true
	if enabled {
		go func() {
			for {
				spawnBackendToFrontend("hyperion5.com")
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// generate Galaxy
	////////////////////////////////////////////////////////////

	enabled = false
	if enabled {
		galaxyType := ELLIPTICAL
		generateGalaxy(galaxyType)
		logInfo(fmt.Sprint("   ...\tDeleting Rtree"))
		rtree = &rtreego.Rtree{}
		rtreeObjects = []rtreego.Spatial{}
	}

	////////////////////////////////////////////////////////////
	// start communication with other Backend servers
	////////////////////////////////////////////////////////////

	enabled = true
	if enabled {
		go kickoffBackendCommunication()
	}

	////////////////////////////////////////////////////////////
	// start handling incoming user requests
	////////////////////////////////////////////////////////////

	GLOBAL.bufferedRequests = []incomingRequest{}
	go func() {
		for {
			GLOBAL.bufferedRequestsMutex.Lock()
			someRequests := make([]incomingRequest, len(GLOBAL.bufferedRequests))
			copy(someRequests, GLOBAL.bufferedRequests)
			GLOBAL.bufferedRequests = []incomingRequest{}
			GLOBAL.bufferedRequestsMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
		}

	}()

	////////////////////////////////////////////////////////////
	// define functions to be used in main game loop
	////////////////////////////////////////////////////////////

	startSendingBufferedRequests := func() {

	}

	copyNewGalaxyToOld := func() {
		x := proto.Clone(&GLOBAL.futureGalaxy)
		GLOBAL.currentGalaxy = *x.(*Galaxy)
		// data, _ := proto.Marshal(&GLOBAL.futureGalaxy)
		// GLOBAL.currentGalaxy = Galaxy{}
		// _ = proto.Unmarshal(data, &GLOBAL.currentGalaxy)
	}

	gatherSystemPtrsForClients := func() {
		GLOBAL.waitForCores["parsedptrs"].Add(GLOBAL.numberOfCores)
		numPlayers := len(GLOBAL.currentGalaxy.GetSolarsystemslice())
		playersPerCore := numPlayers / GLOBAL.numberOfCores
		tempResults := make([]map[string][]*SolarSystem, GLOBAL.numberOfCores)
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := playersPerCore * coreNum
			startIndex := stopIndex - playersPerCore
			go func(i int) {
				tempResults[i] = walkGalaxyForPlayers(startIndex, stopIndex)
				GLOBAL.waitForCores["parsedptrs"].Done()
			}(i)
		}
		GLOBAL.waitForCores["parsedptrs"].Wait()
		GLOBAL.parsedPlayerPtrs = map[string][]*SolarSystem{}
		for i := range tempResults {
			for key, val := range tempResults[i] {
				GLOBAL.parsedPlayerPtrs[key] = val
			}
		}
	}

	////////////////////////////////////////////////////////////
	// primary game loop
	////////////////////////////////////////////////////////////

	GLOBAL.turningDataSignal = make(chan struct{})
	GLOBAL.processingSignal = make(chan struct{})

	go func() {
		for {

			close(GLOBAL.turningDataSignal)
			GLOBAL.turningDataSignal = make(chan struct{})

			startSendingBufferedRequests()

			copyNewGalaxyToOld()
			gatherSystemPtrsForClients()

			close(GLOBAL.processingSignal)
			GLOBAL.processingSignal = make(chan struct{})

			// send galaxyinfo to clients using currentGalaxy data
			//     backend_websocket_handlers

			// calculate galaxy environment changes (pop, battles, movement)
			//     backend_update_galaxy

			// AI rourintes using currentGalaxy to generate bufferedRequests
			//     backend_ai_routine

			// synchronize requests with other backends to move into confirmedRequests
			//     backend_websocket_handlers
		}
	}()

	////////////////////////////////////////////////////////////
	// setup httpMux to avoid leaking default handlers
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	////////////////////////////////////////////////////////////
	// setup the websocket handlers
	////////////////////////////////////////////////////////////

	muxSecure.HandleFunc("/clientgalaxy", handleClientToBackend)
	muxSecure.HandleFunc("/backendsync", handleBackendToBackend)
	muxSecure.HandleFunc("/pingpong", handlePingPongBackend)

	////////////////////////////////////////////////////////////
	// tying the following http parameters to the handlers associated with 'muxSecure'
	////////////////////////////////////////////////////////////

	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         ":443",
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

	////////////////////////////////////////////////////////////
	// serve HTTPS requests with config defined above
	////////////////////////////////////////////////////////////

	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			GLOBAL.waitForLogging.Wait()
			//os.Exit(1)
		}
	}()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffBackendCommunication() {
	GLOBAL.fullBackendStats = []backendInfoStruct{}
	GLOBAL.idleBackendStats = []backendInfoStruct{}
	// https://play.golang.org/p/pFWNe6CGxd

	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	rnum := rand.New(seed())

	for {

		// to avoid contention we are only rlocking and accepting the risk that the
		// information may be out of date and duplicate connections could be spawned
		// > the websocket functions have sanity checks to mitigate.

		GLOBAL.backendDataMutex.RLock()
		potentialServers := []string{}
		copy(potentialServers, GLOBAL.backendData.Backends)
		GLOBAL.backendDataMutex.RUnlock()

		GLOBAL.fullBackendStatsMutex.RLock()
		fullServerConns := make([]string, len(GLOBAL.fullBackendStats))
		for i := 0; i < len(GLOBAL.fullBackendStats); i++ {
			fullServerConns = append(fullServerConns, GLOBAL.fullBackendStats[i].ServerIP)
		}
		GLOBAL.fullBackendStatsMutex.RUnlock()

		GLOBAL.idleBackendStatsMutex.RLock()
		idleServerConns := make([]string, len(GLOBAL.idleBackendStats))
		for i := 0; i < len(GLOBAL.idleBackendStats); i++ {
			idleServerConns = append(idleServerConns, GLOBAL.idleBackendStats[i].ServerIP)
		}
		GLOBAL.idleBackendStatsMutex.RUnlock()

		// we want the list of potentialConnections to not include
		// servers we already have a _full_ connection with
		for i := 0; i < len(fullServerConns); i++ {
			potentialServers = removeStringIfPresent(potentialServers, fullServerConns[i])
		}
		numNewConns := connectedBackendsLimit - len(fullServerConns)
		if numNewConns > 0 {
			randPerm := rnum.Perm(len(potentialServers))
			for i := 0; i < numNewConns; i++ {
				randIndex := randPerm[i]
				randBackendHost := potentialServers[randIndex]
				go spawnBackendToBackend(randBackendHost)
			}
			continue // to get updated info
		}

		// we want the list of potentialConnections to not include
		// servers we already have an _idle_ connection with
		for i := 0; i < len(idleServerConns); i++ {
			potentialServers = removeStringIfPresent(potentialServers, idleServerConns[i])
		}
		numNewConns = idleConnectedBackendsLimit - len(idleServerConns)
		if numNewConns > 0 {
			randPerm := rnum.Perm(len(potentialServers))
			for i := 0; i < numNewConns; i++ {
				randIndex := randPerm[i]
				randBackendHost := potentialServers[randIndex]
				go spawnPingPongBackend(randBackendHost)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func walkGalaxyForPlayers(startIndex, stopIndex int) map[string][]*SolarSystem {
	systemPtrs := map[string][]*SolarSystem{}
	for i := startIndex; i < stopIndex; i++ {
		solarsystem := GLOBAL.currentGalaxy.Solarsystemslice[i]

		addOwner := func(owner string) {
			if owner == "" {
				return
			}
			_, ok := systemPtrs[owner]
			if !ok {
				systemPtrs[owner] = []*SolarSystem{solarsystem}
			} else {
				foundMatch := false
				for _, thisSystem := range systemPtrs[owner] {
					if thisSystem == solarsystem {
						foundMatch = true
					}
				}
				if foundMatch == false {
					systemPtrs[owner] = append(systemPtrs[owner], solarsystem)
				}
			}
		}

		for _, planet := range solarsystem.Planetslice {
			addOwner(planet.GetOwner())
		}
		for _, spaceship := range solarsystem.Spaceshipslice {
			addOwner(spaceship.GetOwner())
		}
		for _, otherobject := range solarsystem.Otherobjectslice {
			addOwner(otherobject.GetOwner())
		}
	}

	return systemPtrs
}

func gracefullyCloseBackend() {

	foundMatch, _ := regexp.MatchString(".*blob$", GLOBAL.outputFile)
	if GLOBAL.outputFile == "" || foundMatch {
		protoOut, err := proto.Marshal(&GLOBAL.currentGalaxy)
		if err != nil {
			log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
		}
		if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	} else {
		jsonOut, err := json.MarshalIndent(GLOBAL.currentGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(GLOBAL.outputFile, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}

	// wait for logging to complete
	logInfo("PROGRAM EXITING GRACEFULLY")
	GLOBAL.waitForLogging.Wait()

	// halt all further execution
	os.Exit(1)

}
