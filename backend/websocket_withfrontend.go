package backend

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/netutil"
	"github.com/gorilla/websocket"
)

// ToFrontendFromBackend ...
type ToFrontendFromBackend struct {
	GalaxyID       int32
	Hostname       string
	GlobalX        int64
	GlobalY        int64
	PlayerStrength map[string]int32
}

// ToBackendFromFrontend ...
type ToBackendFromFrontend struct {
	PlayersToSpawn map[string]int32
}

func (s *service) spawnFrontendSync(url string) error {
	const fName = "(backendToFrontend)"

	dialer := netutil.GetWebsocketDialer(s.args, s.log)
	dialer.Subprotocols = []string{"backend-to-frontend"}
	conn, _, err := dialer.Dial("wss://"+url+"/websocket", nil)
	if err != nil {
		return fmt.Errorf("unable to open socket with %v because %v", url, err)
	}
	defer conn.Close()

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	s.log.Info(fName, "opening connection:", remoteIP, "using", localIP)
	defer s.log.Info(fName, "closing connection:", remoteIP, "using", localIP)

	s.fwall.AddSessionToHistory(remoteIP, fName)
	defer s.fwall.RemoveSessionFromHistory(remoteIP, fName)

	err = auth.KickoffChallengeResponse(fName, conn, s.fwall, s.args, s.log)
	if err != nil {
		return err
	}

	for {
		s.RLock()
		outMsg := ToFrontendFromBackend{
			Hostname: s.args.BackendHostname,
			GalaxyID: s.virtualGalaxy.GetStaticGalaxy().Id,
			GlobalX:  s.virtualGalaxy.GetStaticGalaxy().GlobalX,
			GlobalY:  s.virtualGalaxy.GetStaticGalaxy().GlobalY,
			PlayerStrength: func() map[string]int32 {
				m := map[string]int32{}
				for k, v := range s.playerStrength {
					m[k] = v
				}
				return m
			}(),
		}
		s.RUnlock()

		connWriter, err := conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}
		defer connWriter.Close()

		err = gob.NewEncoder(connWriter).Encode(&outMsg)
		if err != nil {
			return err
		}
		connWriter.Close()

		msgType, connReader, err := conn.NextReader()
		if msgType != websocket.BinaryMessage {
			return fmt.Errorf("%v tried to send a non binary message", remoteIP)
		} else if err != nil {
			return err
		}
		inMsg := ToBackendFromFrontend{}
		err = gob.NewDecoder(connReader).Decode(&inMsg)
		if err != nil {
			return err
		}

		s.Lock()
		for player, strength := range inMsg.PlayersToSpawn {
			s.playersToSpawn[player] = strength
		}
		s.Unlock()

		time.Sleep(time.Second)
	}
}
