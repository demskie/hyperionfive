package frontend

import (
	"hyperionfive/auth"
	"hyperionfive/firewall"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func handleClientToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(clientToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err := fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
	if log.CheckError(err, fName) {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if log.CheckError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "receiving connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	userid := auth.HandleGoogleAuth(fName, conn, fwall, args, log)
	if userid != "" {
		clientToFrontendCommunication(conn, userid)
	}
}

func handleBackendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(backendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	err := fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
	if log.CheckError(err, fName) {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if log.CheckError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "receiving connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	if auth.HandleChallengeResponse(fName, conn, fwall, args, log) {
		handleBackendToFrontendCommunication(conn)
	}
}

type frontendToFrontendMessage struct {
	GalGlobalX     map[string]int64
	GalGlobalY     map[string]int64
	GalClientIDs   map[string][]string
	GalBackends    map[string][]string
	OtherFrontends map[string]time.Time
}

func handleFrontendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(frontendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err := fwall.CheckFirewall(firewall.CHALLENGE, fName, httpRemoteIP) // BUG // CHALLENGE?
	if log.CheckError(err, fName) {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if log.CheckError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "receiving connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	if auth.HandleChallengeResponse(fName, conn, fwall, args, log) {
		frontendToFrontendCommunication(conn)
	}
}

func spawnFrontendToFrontend(host string) {
	const fName = "(frontendToFrontend)"

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/frontendtofrontend", nil)
	if log.CheckError(err, fName, "unable to open websocket with frontend loadbalancer") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "opening connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	if auth.KickoffChallengeResponse(fName, conn, fwall, args, log) {
		frontendToFrontendCommunication(conn)
	}
}
