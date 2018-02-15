package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/golang/protobuf/proto"
)

func startBackendMode() {
	const fName = "(backendMainFunc)"

	if false {
		handleInputParameters(fName)
	}

	if GLOBAL.createGalaxyBool == true && GLOBAL.galaxy.Solarsystemslice == nil {
		galaxyType := ELLIPTICAL
		generateGalaxy(galaxyType)
	}

	if false {
		go func() {
			for {
				spawnBackendToFrontend("hyperion5.com")
				time.Sleep(1 * time.Second)
			}
		}()
	} else {
		// BUG // for debugging only
		GLOBAL.backendData.Lock()
		GLOBAL.backendData.GalaxyName = "TiredArchf24dab"
		GLOBAL.backendData.GlobalX = -1468782671338529121
		GLOBAL.backendData.GlobalY = 6968993604325489026
		if GLOBAL.localAddress == "192.168.99.1" {
			GLOBAL.backendData.Backends = []string{"192.168.99.2"}
		} else if GLOBAL.localAddress == "192.168.99.2" {
			GLOBAL.backendData.Backends = []string{"192.168.99.1"}
		}
		GLOBAL.backendData.Unlock()
	}

	go func() {
		for {
			copyOfBackends := []string{}
			GLOBAL.backendData.RLock()
			copyOfBackends = append(copyOfBackends, GLOBAL.backendData.Backends...)
			GLOBAL.backendData.RUnlock()
			for _, backendAddress := range copyOfBackends {
				spawnBackendToBackend(backendAddress)
			}
			if GLOBAL.debugModeBool {
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	if false {
		for {
			switch {
			case len(GLOBAL.backendData.Frontends) == 0:
				logError(fName, "waiting for fronendSync")
				time.Sleep(1 * time.Second)
				continue
			case GLOBAL.galaxy.Name == "":
				logError(fName, "waiting to be assigned a galaxyName")
				time.Sleep(1 * time.Second)
				continue
			}
			time.Sleep(1 * time.Second)
			break
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////

	serializeSystems := func(systemBytesBuffer map[string]*proto.Buffer) {
		const fName = "(serializeSystems)"
		start := time.Now()
		parallelProcessSystems(GLOBAL.galaxy, func(ss *SolarSystem, coreID int) {
			err := systemBytesBuffer[ss.Name].Marshal(ss)
			checkError(err, fName)
		})
		logTelemetry(fName, fmt.Sprintf("finished in %v milliseconds",
			time.Since(start).Nanoseconds()/1e6))
	}

	getSystemPointers := func(bufferGalaxy *Galaxy) (sysPointers map[string][]*SolarSystem) {
		const fName = "(getSystemPointers)"
		start := time.Now()
		tempSliceMap := make([]map[string][]*SolarSystem, GLOBAL.numberOfCores)
		for i := range tempSliceMap {
			tempSliceMap[i] = make(map[string][]*SolarSystem, maxNumberOfPlayers)
		}
		parallelProcessSystems(bufferGalaxy, func(ss *SolarSystem, coreID int) {
			for _, obj := range ss.Solarobjectslice {
				if obj.Owner == "" {
					continue
				}
				_, ownerExists := tempSliceMap[coreID][obj.Owner]
				if !ownerExists {
					tempSliceMap[coreID][obj.Owner] = []*SolarSystem{ss}
				} else {
					foundMatch := false
					for _, anotherSS := range tempSliceMap[coreID][obj.Owner] {
						if ss == anotherSS {
							foundMatch = true
							break
						}
					}
					if foundMatch == false {
						tempSliceMap[coreID][obj.Owner] = append(
							tempSliceMap[coreID][obj.Owner], ss)
					}
				}
			}
		})
		sysPointers = make(map[string][]*SolarSystem, maxNumberOfPlayers)
		for _, somePointers := range tempSliceMap {
			for k, v := range somePointers {
				sysPointers[k] = v
			}
		}
		logTelemetry(fName, fmt.Sprintf("finished in %v milliseconds",
			time.Since(start).Nanoseconds()/1e6))
		return sysPointers
	}

	////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////////

	go func() {

		// to avoid unncessary allocations we reuse these for mutating
		bufferGalaxy := makeGalaxyBuffer(GLOBAL.galaxy)
		systemBytesBuffer := map[string]*proto.Buffer{}

		// mapping ss.Name to slice index
		sysIndex := map[string]int{}
		for idx, ss := range GLOBAL.galaxy.Solarsystemslice {
			sysIndex[ss.Name] = idx
			systemBytesBuffer[ss.Name] = proto.NewBuffer([]byte{})
		}

		for {
			// record current time and begin timeout tickers
			start := time.Now()
			minimumTimeBlocker := time.NewTimer(time.Second / 10)
			maximumTimeBlocker := time.NewTimer(time.Second * 2)

			// signal that data is now being mutated and is not safe for consumption
			close(GLOBAL.turningDataSignal)
			GLOBAL.turningDataSignal = make(chan struct{})

			// move the galaxy forward into the future
			processGalaxy(bufferGalaxy, sysIndex)

			// serialize systems in the background (this takes a while)
			doneChan := make(chan struct{})
			go func() {
				serializeSystems(systemBytesBuffer)
				close(doneChan)
			}()

			// triggers backend_websockets to send message to neighbors
			close(GLOBAL.backendSignal)
			GLOBAL.backendSignal = make(chan struct{})

			// for every player gather []*SolarSystem
			sysPointers := getSystemPointers(bufferGalaxy)

			// wait for confirmation from every neighbor
			if len(GLOBAL.backendData.Backends) != 0 {
				select {
				case <-GLOBAL.allServersReadySignal:
					logTelemetry(fName, fmt.Sprintf("synchronized in %v milliseconds",
						time.Since(start).Nanoseconds()/1e6))
				case <-maximumTimeBlocker.C:
					logError(fName, "ran out of time to wait for all backends")
				}
			} else {
				// logError(fName, "no other backends are detected")
			}

			// verify bufferGalaxy now that we've received our neighbor's hashes
			verifyGalaxy(bufferGalaxy)

			// wait for the background goroutine to finish
			<-doneChan

			// prepare to serve this new data to clients
			GLOBAL.backendData.Lock()
			GLOBAL.galaxyMutex.Lock()
			GLOBAL.galaxy = bufferGalaxy
			GLOBAL.backendData.ParsedPlayerPtrs = sysPointers
			GLOBAL.galaxyMutex.Unlock()
			GLOBAL.backendData.Unlock()

			// triggers backend_websockets to send message to clients
			close(GLOBAL.clientSignal)
			GLOBAL.clientSignal = make(chan struct{})

			// enforce a maximum tick rate
			<-minimumTimeBlocker.C

			// record the actual tick time
			totalTime := time.Since(start)
			GLOBAL.backendData.Lock()
			GLOBAL.backendData.LocalTickTime = totalTime
			GLOBAL.backendData.Unlock()
			logTelemetry(fName, fmt.Sprintf("finished in %v milliseconds",
				time.Since(start).Nanoseconds()/1e6))

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
				Addr:    getLocalAddr() + ":8080",
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			checkError(errServeInsecure, fName, "(pprof)")
		}()
	}

	// wait for program to signal exit
	<-GLOBAL.exitProgram
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func gracefullyCloseBackend() {
	GLOBAL.galaxyMutex.RLock()
	defer GLOBAL.galaxyMutex.RUnlock()
	foundMatch, _ := regexp.MatchString(".*blob$", GLOBAL.outputFile)
	if GLOBAL.outputFile == "" || foundMatch {
		protoOut, err := proto.Marshal(GLOBAL.galaxy)
		if err != nil {
			log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
		}
		if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	} else {
		jsonOut, err := json.MarshalIndent(GLOBAL.galaxy, "", "  ")
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
