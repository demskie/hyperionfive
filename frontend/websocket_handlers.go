package frontend

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

func (s *Service) handleBackendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(backendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err := fwall.CheckFirewall(firewall.SESSIONLIMIT, fName, httpRemoteIP)
	if log.CheckError(err, fName) {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   0,
		WriteBufferSize:  0,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if log.CheckError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "receiving connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	err = auth.HandleChallengeResponse(fName, conn, fwall, args, log)
	if log.CheckError(err) {
		return
	}
	for {
		err = s.receiveBackendSync(conn)
		if err != nil {
			return
		}
	}
}

func (s *Service) handleFrontendToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(frontendToFrontend)"

	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err := fwall.CheckFirewall(firewall.CHALLENGE, fName, httpRemoteIP) // BUG // CHALLENGE?
	if log.CheckError(err, fName) {
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   0,
		WriteBufferSize:  0,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if log.CheckError(err, fName, "upgrade error") {
		return
	}
	defer conn.Close()

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	log.Info(fName, "receiving connection:", remoteIP, "using", localIP)
	defer log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	fwall.AddSessionToHistory(remoteIP, fName)
	defer fwall.RemoveSessionFromHistory(remoteIP, fName)

	err = auth.HandleChallengeResponse(fName, conn, fwall, args, log)
	if log.CheckError(err, fName) {
		return
	}
	for {
		if s.receiveFrontendSync(conn) != nil {
			return
		}
	}
}

func (s *Service) spawnFrontendToFrontend(host string) error {
	const fName = "(frontendToFrontend)"

	dialer := netutil.GetWebsocketDialer(args, log)
	conn, _, err := dialer.Dial("wss://"+host+":443/frontend-to-frontend", nil)
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
		err = s.startFrontendSync(conn)
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
