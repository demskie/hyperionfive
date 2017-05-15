package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"sync"
	"time"
)

func startFrontendMode() {

	// initialize frontendData
	frontendData = FrontendDataStruct{}

	// setup httpMux to avoid leaking default handlers
	muxSecure := http.NewServeMux()

	// to enable perfect forward secrecy it's necessary to add 'Strict-Transport-Security'
	// and since we are not writing out own FileServer, we'll need to add Headers this way
	addSecurityHeaders := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.ServeHTTP(w, r)
		}
	}
	directory := http.Dir("D:\\OneDrive\\GoPrototypes\\src\\GOSERVER\\websiteroot\\")
	fileserverWithModifiedHeaders := addSecurityHeaders(http.FileServer(directory))

	// setup the fileserver handler now that the necessary security headers have been added
	muxSecure.Handle("/", fileserverWithModifiedHeaders)

	// setup the websocket handlers
	muxSecure.HandleFunc("/clientlb", handleClientToFrontend)
	muxSecure.HandleFunc("/backend", handleBackendToFrontend)
	muxSecure.HandleFunc("/frontend", handleFrontendToFrontend)

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

	// define waitgroup to be used to stop premature termination
	var lbWaitGroup sync.WaitGroup

	// serve HTTPS requests with config defined above
	lbWaitGroup.Add(1)
	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			waitForLogging.Wait()
			//os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// define nonsecure HTTP parameters
	srvInsecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			url := "https://" + req.Host + req.URL.String()
			http.Redirect(w, req, url, http.StatusMovedPermanently)
		}),
	}

	// serve HTTP requests by redirecting to HTTPS
	lbWaitGroup.Add(1)
	go func() {
		errServeInsecure := srvInsecure.ListenAndServe()
		if checkError("(ListenAndServeInsecure) Something went wrong:", errServeInsecure) {
			waitForLogging.Wait()
			//os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// use waitgroup to stop permature termination unless all
	// the above goroutines are finished executing
	lbWaitGroup.Wait()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func gracefullyCloseFrontend() {

	// wait for logging to complete
	logInfo("PROGRAM EXITING GRACEFULLY")
	waitForLogging.Wait()

	// halt all further execution
	os.Exit(1)

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
