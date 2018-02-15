package main

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func handleClientToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(clientToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(SESSIONLIMIT, fName, httpRemoteIP) != true {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "receiving connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	userid := handleGoogleAuth(fName, conn)
	if userid != "" {
		clientToFrontendCommunication(conn, userid)
	}
}

func handleBackendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(backendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(SESSIONLIMIT, fName, httpRemoteIP) != true {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "receiving connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	if handleChallengeResponse(fName, conn) {
		handleBackendToFrontendCommunication(conn)
	}
}

func handleFrontendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(frontendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(CHALLENGE, fName, httpRemoteIP) != true {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "receiving connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	if handleChallengeResponse(fName, conn) {
		frontendToFrontendCommunication(conn)
	}
}

func spawnFrontendToFrontend(host string) {
	const fName = "(frontendToFrontend)"

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/frontendtofrontend", nil)
	if checkError(err, fName, "unable to open websocket with frontend loadbalancer") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1)

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "opening connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	if kickoffChallengeResponse(fName, conn) {
		frontendToFrontendCommunication(conn)
	}
}
