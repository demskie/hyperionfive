package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/golang/protobuf/proto"
)

func startBackendMode() {
	fName := "(backendMainFunc)"

	if true {
		handleInputParameters(fName)
	}

	if true {
		go func() {
			for {
				spawnBackendToFrontend("hyperion5.com")
				time.Sleep(1 * time.Second)
			}
		}()
	}

	if GLOBAL.createGalaxyBool == true {
		galaxyType := ELLIPTICAL
		generateGalaxy(galaxyType)
		logInfo(fmt.Sprint("   ...\tDeleting Rtree"))
		rtree = &rtreego.Rtree{}
		rtreeObjects = []rtreego.Spatial{}
	}

	if true {
		go func() {
			for {
				GLOBAL.backendData.RLock()
				for _, backendAddress := range GLOBAL.backendData.Backends {
					spawnBackendToBackend(backendAddress)
				}
				GLOBAL.backendData.RUnlock()
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	////////////////////////////////////////////////////////////
	// define functions to be used in primary game loop
	////////////////////////////////////////////////////////////

	copyDataFromGlobals := func() {
		GLOBAL.backendData.RLock()
		GLOBAL.futureGalaxy.Name = GLOBAL.backendData.GalaxyName
		GLOBAL.futureGalaxy.GlobalX = GLOBAL.backendData.GlobalX
		GLOBAL.futureGalaxy.GlobalY = GLOBAL.backendData.GlobalY
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
			if checkError(err, fName, "CRITICAL FAILURE - MARSHALLING GALAXY FOR CRC TEST") {
				// TODO // handle this error
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

		seeds := map[string]int64{}
		GLOBAL.backendData.RLock()
		for sysName, sysHash := range GLOBAL.backendData.SystemHashes[0]["0.0.0.0"] {
			seeds[sysName] = int64(binary.LittleEndian.Uint32(sysHash[:]))
		}
		GLOBAL.backendData.RUnlock()

		GLOBAL.bufferedRequestsMutex.Lock()
		requests := []incomingClientRequest{}
		for i, rqst := range GLOBAL.bufferedRequests {
			foundMatch := false
			for _, validID := range validRequestIDs {
				if rqst.UniqueID == validID {
					requests = append(requests, rqst)
					foundMatch = true
					break
				}
			}
			if foundMatch == false {
				GLOBAL.bufferedRequests = append(
					GLOBAL.bufferedRequests[:i],
					GLOBAL.bufferedRequests[i+1:]...)
			}
		}
		GLOBAL.bufferedRequestsMutex.Unlock()

		if len(requests) < len(validRequestIDs) {
			logError("ERROR: missing one or more clientRequests")
		}

		processGalaxy(seeds, requests, GLOBAL.futureGalaxy)
	}

	gatherSystemPtrsForClients := func() {
		wg := sync.WaitGroup{}
		wg.Add(GLOBAL.numberOfCores)
		numPlayers := len(GLOBAL.currentGalaxy.GetSolarsystemslice())
		playersPerCore := numPlayers / GLOBAL.numberOfCores
		tempResults := make([]map[string][]*SolarSystem, GLOBAL.numberOfCores)
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := playersPerCore * coreNum
			startIndex := stopIndex - playersPerCore
			go func(i int) {
				tempResults[i] = walkGalaxyForPlayers(startIndex, stopIndex)
				wg.Done()
			}(i)
		}
		wg.Wait()
		GLOBAL.backendData.Lock()
		GLOBAL.backendData.ParsedPlayerPtrs = map[string][]*SolarSystem{}
		for i := range tempResults {
			for key, val := range tempResults[i] {
				GLOBAL.backendData.ParsedPlayerPtrs[key] = val
			}
		}
		GLOBAL.backendData.Unlock()
	}

	noNeighbors := func() (result bool) {
		GLOBAL.backendData.RLock()
		if len(GLOBAL.backendData.Backends) == 0 {
			result = true
		}
		GLOBAL.backendData.RUnlock()
		return result
	}

	////////////////////////////////////////////////////////////
	// primary game loop
	////////////////////////////////////////////////////////////

	go func() {
		for {

			// record current time and start timeout tickers
			start := time.Now().UnixNano()
			minimumTimeBlocker := time.NewTimer(time.Second / 60)
			maximumTimeBlocker := time.NewTimer(time.Second / 2)

			// signal that data is now being mutated and is not safe for consumption
			close(GLOBAL.turningDataSignal)
			GLOBAL.turningDataSignal = make(chan struct{})
			// an alterate channel for monitoring data mutation
			GLOBAL.mutatingDataBlocker = make(chan struct{})

			// major steps in the gameloop workflow
			copyDataFromGlobals()
			copyFutureIntoCurrentGalaxy()
			generateCurrentSystemHashes()
			processOutstandingUserRequests() // gamelogic goes here
			gatherSystemPtrsForClients()

			// triggers backend_websockets to send message to neighbors
			close(GLOBAL.newTickSignal)
			GLOBAL.newTickSignal = make(chan struct{})
			// an alternate channel for monitoring data mutation
			close(GLOBAL.mutatingDataBlocker)

			if noNeighbors() {
				go func() {
					GLOBAL.backendData.Lock()
					close(GLOBAL.allServersReadySignal)
					//GLOBAL.backendData.ReceivedMessageFrom = []string{"0.0.0.0"} // TODO: remove
					GLOBAL.backendData.Unlock()
				}()
			}

			// wait for confirmation from every neighbor
			select {
			case <-GLOBAL.allServersReadySignal:
				GLOBAL.allServersReadySignal = make(chan struct{})
			case <-maximumTimeBlocker.C:
				logDebug(fName, "ran out of time to wait for backend message")
			}

			// will block if minTime hasn't been met - this enforces a maximum tickRate
			<-minimumTimeBlocker.C

			end := time.Now().UnixNano()
			GLOBAL.backendData.Lock()
			GLOBAL.backendData.TickTime = start - end
			GLOBAL.backendData.Unlock()

		}
	}()

	////////////////////////////////////////////////////////////
	// setup HTTP and HTTPS server
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	// setup the websocket handlers
	muxSecure.HandleFunc("/clienttobackend", handleClientToBackend)
	muxSecure.HandleFunc("/backendtobackend", handleBackendToBackend)

	getLocalAddr := func() (ip string) {
		if GLOBAL.localAddress != "" {
			ip = GLOBAL.localAddress
		} else if GLOBAL.localInterface != "" {
			iface, err := net.InterfaceByName(GLOBAL.localInterface)
			if err != nil {
				logError(fName, GLOBAL.localInterface, "does not match any local interfaces")
				suddenExit()
				return
			}
			addrs, err := iface.Addrs()
			if checkError(err) {
				return
			}
			ip = addrs[0].(*net.IPNet).IP.String()
		}
		if ip != "" {
			logInfo(fName, "serving via the", ip, "address")
		}
		return
	}

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         getLocalAddr() + ":443",
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
		err := http.ListenAndServe(":8080", profmux)
		checkError(err, fName, fmt.Sprintf("Error when starting pprof server:"))
	}

	// wait for program to signal exit
	<-GLOBAL.exitProgram
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

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
		for _, obj := range solarsystem.Solarobjectslice {
			addOwner(obj.GetOwner())
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
	logInfo("PROGRAM EXITING GRACEFULLY")
	os.Exit(1)
}

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
