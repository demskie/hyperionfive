package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
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

	if false {
		handleInputParameters("startBackendMode")
	}

	if true {
		go func() {
			for {
				spawnBackendToFrontend("hyperion5.com")
			}
		}()
	}

	if false {
		galaxyType := ELLIPTICAL
		generateGalaxy(galaxyType)
		logInfo(fmt.Sprint("   ...\tDeleting Rtree"))
		rtree = &rtreego.Rtree{}
		rtreeObjects = []rtreego.Spatial{}
	}

	if true {
		go func() {
			for {
				kickoffBackendCommunication()
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// define functions to be used in primary game loop
	////////////////////////////////////////////////////////////

	checkValidityOfHashes := func() {
		GLOBAL.backendData.RLock()

		GLOBAL.backendData.RUnlock()
	}

	copyFutureIntoCurrentGalaxy := func() {
		x := proto.Clone(&GLOBAL.futureGalaxy)
		GLOBAL.currentGalaxy = *x.(*Galaxy)
	}

	generateCurrentSystemHashes := func() {
		GLOBAL.backendData.Lock()
		for i := range GLOBAL.backendData.SystemHashes {
			x := i - 1
			if x < 0 {
				continue
			}
			for sysName, localHash := range GLOBAL.backendData.SystemHashes[i]["0.0.0.0"] {
				GLOBAL.backendData.SystemHashes[x]["0.0.0.0"][sysName] = localHash
			}
		}
		// https://play.golang.org/p/MkJvT-ngmQ
		newSlice := [16]map[string]map[string][4]byte{}
		for i := 0; i < len(GLOBAL.backendData.SystemHashes)-1; i++ {
			newSlice[i+1] = GLOBAL.backendData.SystemHashes[i]
		}
		for i := range GLOBAL.currentGalaxy.Solarsystemslice {
			data, err := proto.Marshal(GLOBAL.currentGalaxy.Solarsystemslice[i])
			if checkError("CRITICAL FAILURE - MARSHALLING GALAXY FOR CRC TEST: ", err) {
				os.Exit(1)
			}
			systemName := GLOBAL.currentGalaxy.Solarsystemslice[i].Name
			GLOBAL.backendData.SystemBytes[systemName] = data
			buffer := bytes.NewBuffer(data)
			reader := bufio.NewReader(buffer)
			scanner := bufio.NewScanner(reader)
			hash := crc32.NewIEEE()
			for scanner.Scan() {
				hash.Write(scanner.Bytes())
			}
			hashSum := hash.Sum(nil)
			tmpArray := [4]byte{}
			copy(tmpArray[:], hashSum)
			newSlice[0]["0.0.0.0"][systemName] = tmpArray
		}
		copy(GLOBAL.backendData.SystemHashes[:], newSlice[:])
		GLOBAL.backendData.Unlock()
	}

	processOutstandingUserRequests := func() {
		validRequestIDs := []int64{}
		GLOBAL.backendData.Lock()
		for _, allIDs := range GLOBAL.backendData.ConfirmedRequestIDs {
			for _, potentialRequestID := range allIDs {
				validRequest := true
				for neighbor := range GLOBAL.backendData.ConfirmedRequestIDs {
					if validRequest == false {
						break
					}
					foundMatch := false
					for _, id := range GLOBAL.backendData.ConfirmedRequestIDs[neighbor] {
						if id == potentialRequestID {
							foundMatch = true
							break
						}
					}
					if foundMatch == false {
						validRequest = false
					}
				}
				if validRequest == true {
					foundMatch := false
					for _, id := range validRequestIDs {
						if id == potentialRequestID {
							foundMatch = true
						}
					}
					if foundMatch == false {
						validRequestIDs = append(validRequestIDs, potentialRequestID)
					}
				}
			}
		}
		GLOBAL.backendData.ConfirmedRequestIDs = map[string][]int64{}
		GLOBAL.backendData.Unlock()

		// TODO: process these requestIDs
		// TODO: remove these requestIDs from bufferedRequests
		// TODO: remove these requestIDs from confirmedRequests
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
		GLOBAL.backendData.Lock()
		GLOBAL.backendData.ParsedPlayerPtrs = map[string][]*SolarSystem{}
		for i := range tempResults {
			for key, val := range tempResults[i] {
				GLOBAL.backendData.ParsedPlayerPtrs[key] = val
			}
		}
		GLOBAL.backendData.Unlock()
	}

	////////////////////////////////////////////////////////////
	// primary game loop
	////////////////////////////////////////////////////////////

	go func() {
		for {

			// record current time and start timeout tickers
			start := time.Now().UnixNano()
			minimumTimeBlocker := time.NewTimer(time.Second / 30)
			maximumTimeBlocker := time.NewTimer(time.Second / 2)

			// signal that data is now being mutated and is not safe for consumption
			close(GLOBAL.turningDataSignal)
			GLOBAL.turningDataSignal = make(chan struct{})

			// major steps in the gameloop workflow
			checkValidityOfHashes()
			copyFutureIntoCurrentGalaxy()
			generateCurrentSystemHashes()
			processOutstandingUserRequests() // gamelogic here
			gatherSystemPtrsForClients()

			// triggers backend_websockets to send message to neighbors
			close(GLOBAL.newTickSignal)
			GLOBAL.newTickSignal = make(chan struct{})

			// wait for confirmation from every neighbor
			select {
			case <-GLOBAL.allServersReadySignal:
			case <-maximumTimeBlocker.C:
				logError("ERROR: RAN OUT OF TIME TO WAIT FOR BACKEND MESSAGE")
			}

			// will block if minTime hasn't been met - this enforces a maximum tickRate
			<-minimumTimeBlocker.C

			//
			end := time.Now().UnixNano()
			GLOBAL.backendData.TickTime = start - end

		}
	}()

	////////////////////////////////////////////////////////////
	// setup HTTP and HTTPS server
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	// setup the websocket handlers
	muxSecure.HandleFunc("/clientgalaxy", handleClientToBackend)
	muxSecure.HandleFunc("/backendsync", handleBackendToBackend)

	// tying the following http parameters to the handlers associated with 'muxSecure'
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

	// serve HTTPS requests with config defined above
	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			GLOBAL.waitForLogging.Wait()
			os.Exit(1)
		}
	}()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffBackendCommunication() {
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
	// TODO - NEED TO REWRITE - THIS WILL BE FULL-MESH GOING FORWARD
	for {
		// To avoid contention we are only rlocking and accepting the risk that the
		// information may be out of date and duplicate connections could be spawned.
		// Please note, the websocket functions have sanity checks to mitigate.
		GLOBAL.backendData.RLock()
		potentialServers := []string{}
		for serverIP := range GLOBAL.backendData.SystemHashes[0] {
			if serverIP != "0.0.0.0" {
				potentialServers = append(potentialServers, serverIP)
			}
		}
		fullServerConns := []string{}
		for serverIP := range GLOBAL.backendData.Latencies {
			fullServerConns = append(fullServerConns, serverIP)
		}
		GLOBAL.backendData.RUnlock()
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

func recordReceiptOfMessage(remoteIPAddr string) {
	foundMatch := false
	for _, neigh := range GLOBAL.backendData.ReceivedMessageFrom {
		if neigh == remoteIPAddr {
			foundMatch = true
			break
		}
	}
	if foundMatch == false {
		GLOBAL.backendData.ReceivedMessageFrom = append(GLOBAL.backendData.ReceivedMessageFrom,
			remoteIPAddr)
	}
	missedMatch := false
	for _, knownBackend := range GLOBAL.backendData.Backends {
		foundMatch = false
		for _, aliveBackend := range GLOBAL.backendData.ReceivedMessageFrom {
			if knownBackend == aliveBackend {
				foundMatch = true
				break
			}
		}
		if foundMatch == false {
			missedMatch = true
			break
		}
	}
	if missedMatch == false {
		close(GLOBAL.allServersReadySignal)
		GLOBAL.allServersReadySignal = make(chan struct{})
		GLOBAL.backendData.ReceivedMessageFrom = []string{"0.0.0.0"}
	}
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
