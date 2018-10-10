package backend

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/netutil"

	"github.com/gorilla/websocket"
)

func (universe *UniverseStruct) handleClientToBackend(w http.ResponseWriter, r *http.Request) {
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

	userID, err := auth.HandleGoogleAuth(fName, conn, nil, fwall, args, log)
	if err != nil {
		universe.ClientToBackendCommunication(conn, userID)
	}
}

func (universe *UniverseStruct) spawnBackendToFrontend(host string) error {
	const fName = "(backendToFrontend)"

	dialer := netutil.GetWebsocketDialer(args, log)
	conn, _, err := dialer.Dial("wss://"+host+":443/backend-to-frontend", nil)
	if err != nil {
		return fmt.Errorf("unable to open socket with %v because %v", host, err)
	}
	defer conn.Close()

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "opening connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	err = auth.KickoffChallengeResponse(fName, conn, fwall, args, log)
	if err != nil {
		return err
	}
	for {
		err = universe.startFrontendSync(conn)
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
