package backend

import (
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alecthomas/units"
	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/backend/counters"
	"github.com/demskie/hyperionfive/backend/subdivision"
	"github.com/demskie/hyperionfive/backendmessage"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/frontendmessage"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/demskie/logging"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"

	// runtimepprof "runtime/pprof"
	_ "net/http/pprof"
)

type service struct {
	sync.RWMutex
	args                *arguments.Args
	log                 *logging.Logger
	fwall               *firewall.Checker
	googleAuth          *auth.GoogleAuth
	virtualGalaxy       *subdivision.VirtualizedGalaxy
	perfStats           *counters.PerfStats
	rateLimiter         *counters.RateLimiter
	timeMultiplier      float64
	pendingPlayerSpawns map[string]int32
	playerStrength      map[string]int32
	playersToSpawn      map[string]int32
}

func newService(ar *arguments.Args, lg *logging.Logger) *service {
	fw := firewall.New()
	ga := auth.NewGoogleAuth(lg, fw, ar.GoogleAudience)
	return &service{
		args:                ar,
		log:                 lg,
		fwall:               fw,
		googleAuth:          ga,
		virtualGalaxy:       nil,
		perfStats:           counters.NewPerfStats(lg),
		rateLimiter:         counters.NewRateLimiter(),
		timeMultiplier:      1,
		pendingPlayerSpawns: map[string]int32{},
		playerStrength:      map[string]int32{},
		playersToSpawn:      map[string]int32{},
	}
}

// Start ...
func Start(arguments *arguments.Args, logger *logging.Logger) {
	const fName = "(backend)"

	s := newService(arguments, logger)
	s.log.Info(fName, fmt.Sprintf("generating %v systems",
		intutil.CommaFormatNum(backendmessage.TotalNumSystems)))

	s.Lock()
	go func() {
		for {
			err := s.spawnFrontendSync("hyperion5.com")
			s.log.CheckError(err, fName, "failed frontend connection >")
			time.Sleep(1 * time.Second)
		}
	}()

	s.virtualGalaxy = subdivision.NewVirtualizedGalaxy(s.args, 256)
	s.Unlock()

	// continuously process galaxy
	go func() {
		for {
			s.timeMultiplier = s.rateLimiter.Update()

			if s.timeMultiplier >= 1 {
				s.log.Error(fName, fmt.Sprintf("iteration took %v seconds", s.timeMultiplier))
			}

			s.Lock()
			// DO SOME WORK
			s.Unlock()
		}
	}()

	staticGalaxyHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
			b := s.virtualGalaxy.GetStaticGalaxyBytes()
			copiedBytes := make([]byte, len(b))
			copy(copiedBytes, b)
			s.RUnlock()
			_, err = w.Write(copiedBytes)
			s.log.CheckError(err, fName, "failed to completely write response")
		}
	}

	websocketHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		origin := r.Header.Get("Origin")
		if origin != "http://localhost:3000" {
			origin = "https://hyperion5.com"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
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
				"client-to-backend",
			},
			CheckOrigin:       func(r *http.Request) bool { return true },
			EnableCompression: false,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if s.log.CheckError(err, fName, "upgrade error") {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(1 * int64(units.Megabyte))
		userID, err := s.googleAuth.HandleWebsocket(fName, conn)
		if s.log.CheckError(err, fName, httpRemoteIP) {
			return
		}
		switch conn.Subprotocol() {
		case "client-to-backend":
			s.clientToBackend(conn, userID)
		default:
			s.log.Info(fName, "received invalid Subprotocol:", conn.Subprotocol())
		}
	}

	router := httprouter.New()
	router.POST("/staticgalaxy", staticGalaxyHandler)
	router.GET("/websocket", websocketHandler)

	srvSecure := &http.Server{
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
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
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
		errServeTLS := srvSecure.ListenAndServeTLS(
			filepath.Join(dir, "server.rsa.crt"),
			filepath.Join(dir, "server.rsa.key"))
		if s.log.CheckError(errServeTLS, fName) {
			s.log.SuddenExit()
		}
	}()

	// serve HTTP requests just for Debug purposes
	if s.args.DebugModeBool {
		// go get -u github.com/google/pprof
		// pprof -http=":" %GOPATH%/bin/hyperionfive.exe http://192.168.1.70:8080/debug/pprof/profile
		// pprof -http=":" %GOPATH%/bin/hyperionfive.exe http://192.168.1.70:8080/debug/pprof/heap
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
				Addr:    fmt.Sprintf("[%v]:8080", netutil.GetLocalAddress(s.args, s.log)),
				Handler: profmux,
			}
			errServeInsecure := srvProf.ListenAndServe()
			s.log.CheckError(errServeInsecure, fName, "(net/http/pprof)")
		}()
	}

	<-make(chan struct{})
}

func distanceBetweenPoints(x1, y1, x2, y2 int64) uint64 {
	return uint64(math.Hypot(float64(x1)-float64(x2), float64(y1)-float64(y2)))
}
