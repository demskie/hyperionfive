package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func handleClientToBackend(w http.ResponseWriter, r *http.Request) {
	const fName = "(clientToBackend)"

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
		clientToBackendCommunication(conn, userid)
	}
}

func spawnBackendToFrontend(host string) {
	const fName = "(backendToFrontend)"

	dialer := getWebsocketDialer()
	conn, _, err := dialer.Dial("wss://"+host+":443/backendtofrontend", nil)
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
		kickoffBackendToFrontendCommunication(conn)
	}
}

func spawnBackendToBackend(host string) {
	const fName = "(backendToBackend)"

	if checkForExistingConnection("", host) {
		return
	}

	primaryLocalIP := getLocalAddress().IP.String()
	if primaryLocalIP > host {
		logTelemetry(fName, "not opening backendToBackend connection because", primaryLocalIP, "is greater than", host)
		return
	}

	dialer := getWebsocketDialer()
	dialer.ReadBufferSize = 4096 * 4
	dialer.WriteBufferSize = 4096 * 4
	dialer.EnableCompression = true
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // handleChallengeResponse is handling authentication
		//ServerName: host, 	  // doesn't seem to help
	}
	conn, _, err := dialer.Dial("wss://"+host+":443/backendtobackend", nil)
	if checkError(err, fName, "unable to open websocket with backend") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1) // 1 is default

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "opening connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	if kickoffChallengeResponse(fName, conn) {
		backendToBackendCommunication(conn)
	}
}

func handleBackendToBackend(w http.ResponseWriter, r *http.Request) {
	const fName = "(backendToBackend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if checkFirewall(CHALLENGE, fName, httpRemoteIP) != true {
		return
	}

	primaryLocalIP := getLocalAddress().IP.String()
	if httpRemoteIP > primaryLocalIP {
		logError(fName, "unexpected connection from", httpRemoteIP, "to", primaryLocalIP,
			"because", httpRemoteIP, "is greater than", primaryLocalIP)
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout:  30 * time.Second,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
		EnableCompression: true,
		//CheckOrigin: func(r *http.Request) bool {
		//	return true
		//},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(1) // 1 is default

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	logInfo(fName, "receiving connection:", remoteIP, "using", localIP)
	defer logInfo(fName, "closing connection:", remoteIP, "using", localIP)

	addSessionToHistory(remoteIP, fName)
	defer removeSessionFromHistory(remoteIP, fName)

	if handleChallengeResponse(fName, conn) {
		backendToBackendCommunication(conn)
	}
}
