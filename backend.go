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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startBackendMode() {

	// initialize primary data structures
	backendData = BackendDataStruct{}
	currentGalaxy = Galaxy{}
	futureGalaxy = Galaxy{}

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
	// process CLI parameters to potentially import data
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
				kickoffBackendToFrontend("hyperion5.com")
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

	go func() {
		connectedBackends := []Backend{}
		idleConnectedBackends := []Backend{}

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

			backendDataMutex.RLock()
			copyOtherBackends := backendData.Backends
			backendDataMutex.RUnlock()

			connectedBackendsMutex.RLock()
			copyConnectedBackends := connectedBackends
			connectedBackendsMutex.RUnlock()
			numConnectedBackends := len(copyConnectedBackends)

			idleConnectedBackendsMutex.RLock()
			copyIdleConnectedBackends := idleConnectedBackends
			idleConnectedBackendsMutex.RUnlock()
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
					go kickoffIdleBackendToBackend(randBackendHost)
				}
			}

			if numConnectedBackends < connectedBackendsLimit {
				numNewConnections := connectedBackendsLimit - numConnectedBackends
				randPerm := rnum.Perm(numOtherBackends)
				for i := 0; i < numNewConnections; i++ {
					randIndex := randPerm[i]
					randBackendHost := copyOtherBackends[randIndex]
					go kickoffBackendToBackend(randBackendHost)
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	////////////////////////////////////////////////////////////
	// setup httpMux to avoid leaking default handlers
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	////////////////////////////////////////////////////////////
	// setup the websocket handlers
	////////////////////////////////////////////////////////////

	muxSecure.HandleFunc("/clientgalaxy", clientToBackend)
	muxSecure.HandleFunc("/backendsync", backendToBackend)

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
			waitForLogging.Wait()
			//os.Exit(1)
		}
	}()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func gracefullyCloseBackend() {

	foundMatch, _ := regexp.MatchString(".*blob$", outputFilename)
	if outputFilename == "" || foundMatch {
		protoOut, err := proto.Marshal(&currentGalaxy)
		if err != nil {
			log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
		}
		if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	} else {
		jsonOut, err := json.MarshalIndent(currentGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(outputFilename, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}

	// wait for logging to complete
	logInfo("PROGRAM EXITING GRACEFULLY")
	waitForLogging.Wait()

	// halt all further execution
	os.Exit(1)

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
