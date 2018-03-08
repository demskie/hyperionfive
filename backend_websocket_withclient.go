package main

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

func clientToBackendCommunication(conn *websocket.Conn, authenticatedUserID string) {
	fName := "(clientToBackend)"

	////////////////////////////////////////////////////////////
	// create channel to backout of nested gofuncs and for loops
	////////////////////////////////////////////////////////////

	backoutSignal := make(chan struct{})
	var backoutOnce sync.Once
	killFunc := func() {
		close(backoutSignal)
	}
	defer backoutOnce.Do(killFunc)

	////////////////////////////////////////////////////////////
	// reused objects
	////////////////////////////////////////////////////////////

	protoBuffer := &proto.Buffer{}

	////////////////////////////////////////////////////////////
	// receive request from client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				var inMsg struct {
					ClientMessage string   `json:"clientMessage"`
					ClientValues  []string `json:"clientValues"`
				}
				err := conn.ReadJSON(&inMsg)
				if checkError(err, fName, "error receiving request from Client") {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName, "received the following JSON client request:\n",
					fmt.Sprintf("%v\n", inMsg))
				GLOBAL.clientRequests.Lock()
				GLOBAL.clientRequests.Data = append(GLOBAL.clientRequests.Data,
					modifiedClientRequest{
						ClientID:      authenticatedUserID,
						ClientMessage: inMsg.ClientMessage,
						ClientValues:  inMsg.ClientValues,
					})
				GLOBAL.clientRequests.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// send data to client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			// zeroise buffer
			protoBuffer.Reset()

			// wait for new tick
			<-GLOBAL.clientSignal

			// block global mutation
			GLOBAL.galaxyInfo.RLock()

			// prepare data
			customGalaxy := &Galaxy{
				Name:             GLOBAL.galaxyInfo.GalaxyName,
				GlobalX:          GLOBAL.galaxyInfo.GlobalX,
				GlobalY:          GLOBAL.galaxyInfo.GlobalY,
				Solarsystemslice: GLOBAL.galaxyInfo.PlayerPtrs[authenticatedUserID],
			}

			// serialize this information
			err := protoBuffer.Marshal(customGalaxy)
			if checkError(err, fName, "unable to marshal playerData") {
				continue
			}

			// unblock global mutuation
			GLOBAL.galaxyInfo.RLock()

			err = conn.WriteMessage(websocket.BinaryMessage, protoBuffer.Bytes())
			if checkError(err, fName, "error sending message to client") {
				backoutOnce.Do(killFunc)
				return
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// do not finish terminating unless we receive signal
	////////////////////////////////////////////////////////////

	<-backoutSignal
	return
}
