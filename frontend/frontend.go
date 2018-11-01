package frontend

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alecthomas/units"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/frontendmessage"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/demskie/logging"
)

type service struct {
	sync.RWMutex
	args            *arguments.Args
	log             *logging.Logger
	fwall           *firewall.Checker
	googleAuth      *auth.GoogleAuth
	dynamicUniverse *proto.Buffer
	instanceName    string
	version         int32
	precision       int32
	allServers      map[int32]*frontendmessage.Backend
	playersToSpawn  map[string]int32
}

func newService(ar *arguments.Args, lg *logging.Logger) *service {
	fw := firewall.New()
	ga := auth.NewGoogleAuth(lg, fw, ar.GoogleAudience)
	return &service{
		args:            ar,
		log:             lg,
		fwall:           fw,
		googleAuth:      ga,
		dynamicUniverse: &proto.Buffer{},
		instanceName:    "alpha",
		version:         0,
		precision:       256,
		allServers:      make(map[int32]*frontendmessage.Backend, 0),
		playersToSpawn:  make(map[string]int32, 0),
	}
}

// Start ...
func Start(arguments *arguments.Args, logger *logging.Logger) {
	const fName = "(frontend)"

	s := newService(arguments, logger)
	s.log.Info(fName, "starting")

	dynamicUniverseHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		origin := r.Header.Get("Origin")
		if origin != "http://localhost:3000" {
			origin = "https://hyperion5.com"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/octet-stream")
		httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		err := s.fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
		if s.log.CheckError(err, fName, httpRemoteIP) {
			return
		}
		s.fwall.AddSessionToHistory(httpRemoteIP, fName)
		defer s.fwall.RemoveSessionFromHistory(httpRemoteIP, fName)
		_, err = s.googleAuth.HandleRestful(fName, r)
		if s.log.CheckError(err, fName, httpRemoteIP) {
			b, err := proto.Marshal(
				&frontendmessage.ServerInfo{
					ErrorMessage: err.Error(),
				},
			)
			if s.log.CheckError(err, fName, "unable to marshal error message") {
				return
			}
			w.Write(b)
		} else {
			s.RLock()
			b := s.dynamicUniverse.Bytes()
			copiedBytes := make([]byte, len(b))
			copy(copiedBytes, b)
			s.RUnlock()
			_, err = w.Write(copiedBytes)
			s.log.CheckError(err, fName, "failed to completely write response")
		}
	}

	websocketHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		err := s.fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
		if s.log.CheckError(err, fName, httpRemoteIP) {
			return
		}
		s.fwall.AddSessionToHistory(httpRemoteIP, fName)
		defer s.fwall.RemoveSessionFromHistory(httpRemoteIP, fName)
		upgrader := websocket.Upgrader{
			HandshakeTimeout: 15 * time.Second,
			ReadBufferSize:   0,
			WriteBufferSize:  0,
			Subprotocols: []string{
				"backend-to-frontend",
				"frontend-to-frontend",
			},
			EnableCompression: false,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if s.log.CheckError(err, fName, "upgrade error") {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(1 * int64(units.Megabyte))
		err = auth.HandleChallengeResponse(fName, conn, s.fwall, s.args, s.log)
		if err != nil {
			return
		}
		switch conn.Subprotocol() {
		case "backend-to-frontend":
			for {
				err = s.receiveBackendSync(conn)
				if s.log.CheckError(err, fName) {
					return
				}
			}
		case "frontend-to-frontend":
			for {
				err = s.receiveFrontendSync(conn)
				if s.log.CheckError(err, fName) {
					return
				}
			}
		default:
			s.log.Info(fName, fmt.Sprintf("received invalid Subprotocol: '%v'", conn.Subprotocol()))
		}
	}

	router := httprouter.New()
	router.POST("/dynamic-universe", dynamicUniverseHandler)
	router.GET("/websocket", websocketHandler)
	router.NotFound = http.FileServer(http.Dir(filepath.Dir(`C:\Users\alexd\OneDrive\react-hyperion5-game\build\`)))

	secureServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         fmt.Sprintf("[%v]:443", netutil.GetLocalAddress(s.args, s.log)),
		Handler:      router,
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
		TLSNextProto: nil,
	}

	go func() {
		exPath, err := os.Executable()
		if s.log.CheckError(err, fName) {
			s.log.SuddenExit()
		}
		dir := filepath.Dir(exPath)
		errServeTLS := secureServer.ListenAndServeTLS(
			filepath.Join(dir, "server.rsa.crt"),
			filepath.Join(dir, "server.rsa.key"))
		if s.log.CheckError(errServeTLS, fName) {
			s.log.SuddenExit()
		}
	}()

	if s.args.DebugModeBool {
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
				Addr:    netutil.GetLocalAddress(s.args, s.log) + ":8080",
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			s.log.CheckError(errServeInsecure, fName, "(pprof)")
		}()
	} else {

		// define nonsecure HTTP parameters
		srvInsecure := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			Addr:         netutil.GetLocalAddress(s.args, s.log) + ":80",
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
			if s.log.CheckError(errServeInsecure, fName) {
				s.log.SuddenExit()
			}
		}()
	}

	s.log.Info("(hyperionfive) serving HTTPS", netutil.GetLocalAddress(s.args, s.log)+":443")
	<-make(chan struct{})

	return
}
