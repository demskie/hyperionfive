package backend

import (
	"net"
	"net/http"
	"time"

	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/netutil"

	"github.com/gorilla/websocket"
)

// HandleClientToBackend ...
func (universe *UniverseStruct) HandleClientToBackend(w http.ResponseWriter, r *http.Request) {
	const fName = "(clientToBackend)"

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
		id := universe.PlayerNameData.GetReferenceInt(userid)
		universe.ClientToBackendCommunication(conn, id)
	}
}

// SpawnBackendToFrontend ...
func (universe *UniverseStruct) SpawnBackendToFrontend(host string, galaxy *GalaxyStruct) {
	const fName = "(backendToFrontend)"

	dialer := netutil.GetWebsocketDialer(args, log)
	conn, _, err := dialer.Dial("wss://"+host+":443/backendtofrontend", nil)
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
		universe.KickoffBackendToFrontendCommunication(conn, galaxy)
	}
}
