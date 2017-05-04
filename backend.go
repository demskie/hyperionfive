package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

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
	// process CLI parameters to potentially import data
	////////////////////////////////////////////////////////////

	enabled = false
	if enabled {
		handleInputParameters("startBackendMode")
	}

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	enabled = true
	if enabled {
		go func() {
			for {
				kickoffFrontendInfo("hyperion5.com")
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
	// setup httpMux to avoid leaking default handlers
	////////////////////////////////////////////////////////////

	muxSecure := http.NewServeMux()

	////////////////////////////////////////////////////////////
	// setup the websocket handlers
	////////////////////////////////////////////////////////////

	muxSecure.HandleFunc("/clientgalaxy", backendReceiveClientGalaxyRequest)
	muxSecure.HandleFunc("/backendsyncinfo", backendReceiveBackendInfoRequest)
	muxSecure.HandleFunc("/backendsyncgalaxy", backendReceiveBackendGalaxyRequest)

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

	foundMatch, _ := regexp.MatchString(".*blob$", inputFilename)
	if inputFilename == "" || foundMatch {
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
