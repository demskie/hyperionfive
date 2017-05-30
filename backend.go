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

	// easy on/off handle for debugging purposes
	var enabled bool

	////////////////////////////////////////////////////////////
	// generate randomNumber to determine server priority
	////////////////////////////////////////////////////////////

	enabled = true
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
	// game loop
	////////////////////////////////////////////////////////////

	go func() {
		GLOBAL.tickChan = make(chan struct{})

		GLOBAL.currentGalaxy = GLOBAL.futureGalaxy

		GLOBAL.waitForCores.Add(GLOBAL.numberOfCores)
		numPlayers := len(GLOBAL.currentGalaxy.GetSolarsystemslice())
		playersPerCore := numPlayers / GLOBAL.numberOfCores
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := playersPerCore * coreNum
			startIndex := stopIndex - playersPerCore
			go func() {
				result := walkGalaxyForPlayers(startIndex, stopIndex)
			}()
		}
		GLOBAL.waitForCores.Wait()

		close(GLOBAL.tickChan)
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

// Copypasta'd from:
// https://rogpeppe.wordpress.com/2009/12/01/concurrent-idioms-1-broadcasting-values-in-go-with-linked-channels/
// https://play.golang.org/p/HCbY04zIg3

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffBackendCommunication() {
	GLOBAL.connectedBackends = []backend{}
	GLOBAL.idleConnectedBackends = []backend{}
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

		GLOBAL.backendDataMutex.RLock()
		copyOtherBackends := GLOBAL.backendData.Backends
		GLOBAL.backendDataMutex.RUnlock()

		GLOBAL.connectedBackendsMutex.RLock()
		copyConnectedBackends := GLOBAL.connectedBackends
		GLOBAL.connectedBackendsMutex.RUnlock()
		numConnectedBackends := len(copyConnectedBackends)

		GLOBAL.idleConnectedBackendsMutex.RLock()
		copyIdleConnectedBackends := GLOBAL.idleConnectedBackends
		GLOBAL.idleConnectedBackendsMutex.RUnlock()
		numIdleConnectedBackends := len(copyIdleConnectedBackends)

		// we want the list of reference backends to not include Servers we're already connected to
		for i := 0; i < numConnectedBackends; i++ {
			copyOtherBackends = removeStringIfPresent(copyOtherBackends, copyConnectedBackends[i].ServerIP)
		}
		numOtherBackends := len(copyOtherBackends)

		// we want a list of backends that do not include Servers we have an idle connection with
		copyOtherBackendsNotIdle := copyOtherBackends
		for i := 0; i < numConnectedBackends; i++ {
			copyOtherBackendsNotIdle = removeStringIfPresent(copyOtherBackendsNotIdle,
				copyConnectedBackends[i].ServerIP)
		}
		numOtherBackendsNotIdle := len(copyOtherBackendsNotIdle)

		if numIdleConnectedBackends < idleConnectedBackendsLimit {
			numNewIdleConnections := idleConnectedBackendsLimit - numIdleConnectedBackends
			randPerm := rnum.Perm(numOtherBackendsNotIdle)
			for i := 0; i < numNewIdleConnections; i++ {
				randIndex := randPerm[i]
				randBackendHost := copyOtherBackendsNotIdle[randIndex]
				go spawnPingPongBackend(randBackendHost)
			}
			time.Sleep(1000 * time.Millisecond)
		}

		if numConnectedBackends < connectedBackendsLimit {
			numNewConnections := connectedBackendsLimit - numConnectedBackends
			randPerm := rnum.Perm(numOtherBackends)
			for i := 0; i < numNewConnections; i++ {
				randIndex := randPerm[i]
				randBackendHost := copyOtherBackends[randIndex]
				go spawnBackendToBackend(randBackendHost)
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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

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
