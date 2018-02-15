package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

func clientToBackendCommunication(conn *websocket.Conn, authenticatedUserID string) {
	fName := "(clientToBackend)"
	//remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	//localIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	var err error // satisfying go vet

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

	// nothing here

	////////////////////////////////////////////////////////////
	// receive requests from client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			select {
			case <-backoutSignal:
				return
			default:
				inMsg := originalClientRequest{}
				err = conn.ReadJSON(&inMsg)

				if checkError(err, fName, "error receiving request from Client") {
					backoutOnce.Do(killFunc)
					return
				}
				logDebug(fName, "received the following JSON client request:\n",
					fmt.Sprintf("%v\n", inMsg))

				GLOBAL.allRequests.Lock()
				GLOBAL.allRequests.Buffered[<-GLOBAL.uniqueRequestID] =
					modifiedClientRequest{
						ClientID:      authenticatedUserID,
						ClientMessage: inMsg.ClientMessage,
						ClientValues:  inMsg.ClientValues,
						ReceivedTime:  time.Now(),
					}
				GLOBAL.allRequests.Unlock()
			}
		}
	}()

	////////////////////////////////////////////////////////////
	// send data to client
	////////////////////////////////////////////////////////////

	go func() {
		for {
			data := []byte{}
			dataReadySignal := make(chan struct{})
			continueSignal := make(chan struct{})
			go func() {
				// wait for new tick
				<-GLOBAL.clientSignal

				// put together data
				playerData := Galaxy{}
				GLOBAL.backendData.RLock()
				if len(GLOBAL.backendData.SystemsInRecovery) != 0 {
					playerData.Name = GLOBAL.backendData.GalaxyName
					playerData.GlobalX = GLOBAL.backendData.GlobalX
					playerData.GlobalY = GLOBAL.backendData.GlobalY
					playerData.Solarsystemslice = append(playerData.Solarsystemslice,
						GLOBAL.backendData.ParsedPlayerPtrs[authenticatedUserID]...)
				}
				GLOBAL.backendData.RUnlock()

				// serialize this information
				data, err = proto.Marshal(&playerData)
				if checkError(err, fName, "unable to marshal playerdata") {
					close(continueSignal)
				}

				// signal that data is ready
				close(dataReadySignal)
			}()

			select {
			case <-dataReadySignal:
				err = conn.WriteMessage(websocket.BinaryMessage, data)
				if checkError(err, fName, "error sending message to client") {
					backoutOnce.Do(killFunc)
					return
				}
			case <-GLOBAL.turningDataSignal:
				logError(fName, "CRITICAL FAILURE - RAN OUT OF TIME TO SEND PLAYERDATA")
				backoutOnce.Do(killFunc)
				return
			case <-continueSignal:
				continue
			case <-backoutSignal:
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
