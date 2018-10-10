package frontend

import (
	"crypto/tls"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/demskie/logging"
)

var (
	args  *arguments.Args
	log   *logging.Logger
	fwall *firewall.Checker
)

// Start ...
func Start(arguments *arguments.Args, logger *logging.Logger) {
	const fName = "(frontend)"

	args = arguments
	log = logger
	fwall = firewall.New()

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
	directoryObject := http.Dir(filepath.Dir(`C:\Users\alexd\OneDrive\react-hyperion5-game\build\`))
	fileserverWithModifiedHeaders := addSecurityHeaders(http.FileServer(directoryObject))

	// initialize service
	service := &Service{}

	// link the http handlers
	muxSecure.HandleFunc("/client-to-frontend", service.handleClientToFrontend)
	muxSecure.HandleFunc("/backend-to-frontend", service.handleBackendToFrontend)
	muxSecure.HandleFunc("/frontend-to-frontend", service.handleFrontendToFrontend)

	// link the fileserver handler now that the necessary security headers have been added
	muxSecure.Handle("/", fileserverWithModifiedHeaders)

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         netutil.GetLocalAddress(args, log) + ":443",
		Handler:      muxSecure,

		// defining various security parameters used to specify accepted ciphersuites
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},

		// enables HTTP2
		TLSNextProto: nil,
	}

	// serve HTTPS requests with config defined above
	go func() {
		exPath, err := os.Executable()
		if log.CheckError(err, fName) {
			log.SuddenExit()
		}
		dir := filepath.Dir(exPath)
		errServeTLS := srvSecure.ListenAndServeTLS(
			filepath.Join(dir, "server.rsa.crt"),
			filepath.Join(dir, "server.rsa.key"))
		if log.CheckError(errServeTLS, fName) {
			log.SuddenExit()
		}
	}()

	if args.DebugModeBool {
		go func() {
			// serve HTTP requests just for Debug purposes
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
			srvProf := &http.Server{
				Addr:    netutil.GetLocalAddress(args, log) + ":8080",
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			log.CheckError(errServeInsecure, fName, "(pprof)")
		}()
	} else {

		// define nonsecure HTTP parameters
		srvInsecure := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			Addr:         netutil.GetLocalAddress(args, log) + ":80",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Connection", "close")
				url := "https://" + req.Host + req.URL.String()
				http.Redirect(w, req, url, http.StatusMovedPermanently)
			}),
		}
		// no need for keepalives on a handler that only redirects
		srvInsecure.SetKeepAlivesEnabled(false)

		// serve HTTP requests by redirecting to HTTPS
		go func() {
			errServeInsecure := srvInsecure.ListenAndServe()
			if log.CheckError(errServeInsecure, fName) {
				log.SuddenExit()
			}
		}()
	}

	log.Info("(hyperionfive) serving HTTPS", netutil.GetLocalAddress(args, log)+":443")
	<-make(chan struct{})

	return
}
