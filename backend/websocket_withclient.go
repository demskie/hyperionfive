package backend

import (
	"fmt"
	"sync"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

// ClientToBackendCommunication ...
func (universe *UniverseStruct) ClientToBackendCommunication(conn *websocket.Conn, authenticatedUserID uint64) {
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

	// BUG: when receiving data from client, use a channel or slice to add
	// data to universe so that we don't have block a consecutive request

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
				if log.CheckError(err, fName, "error receiving request from Client") {
					backoutOnce.Do(killFunc)
					return
				}
				log.Debug(fName, "received the following JSON client request:\n",
					fmt.Sprintf("%v\n", inMsg))
				// TODO // not implemented
				//	GLOBAL.clientRequests.Lock()
				//	GLOBAL.clientRequests.Data = append(GLOBAL.clientRequests.Data,
				//		modifiedClientRequest{
				//			ClientID:      authenticatedUserID,
				//			ClientMessage: inMsg.ClientMessage,
				//			ClientValues:  inMsg.ClientValues,
				//		})
				//	GLOBAL.clientRequests.Unlock()
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
			<-universe.ClientSignal

			// block global mutation
			universe.RLock()

			// prepare data
			partialGalaxy := &gamestate.Galaxy{
				NameUint:    universe.GalaxyStruct.Protobuf.NameUint,
				GlobalX:     universe.GalaxyStruct.Protobuf.GlobalX,
				GlobalY:     universe.GalaxyStruct.Protobuf.GlobalY,
				Systemslice: universe.GalaxyStruct.PlayerFogOfWarSlice(authenticatedUserID),
			}

			// serialize this information
			err := protoBuffer.Marshal(partialGalaxy)
			if log.CheckError(err, fName, "unable to marshal playerData") {
				continue
			}

			// unblock global mutuation
			universe.RLock()

			err = conn.WriteMessage(websocket.BinaryMessage, protoBuffer.Bytes())
			if log.CheckError(err, fName, "error sending message to client") {
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
