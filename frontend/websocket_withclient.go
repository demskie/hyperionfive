package frontend

import "github.com/gorilla/websocket"

func clientToFrontendCommunication(conn *websocket.Conn, userid string) {
	//fName := "(clientToFrontend)"
	//remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// send list of galaxy servers to client
	////////////////////////////////////////////////////////////

	type PotentialServers struct {
		GalaxyName string
		ServerList []string
	}
	type outgoingMessage struct {
		Servers []PotentialServers
	}

	////////////////////////////////////////////////////////////
	//
	////////////////////////////////////////////////////////////

}
