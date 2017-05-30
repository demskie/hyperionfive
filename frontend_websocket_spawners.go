package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/gorilla/websocket"
)

func spawnFrontendToFrontend(host string) {

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError("(handshakeFrontend) unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	logDebug("(handshakeFrontend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())

	if kickoffChallengeResponse("handshakeFrontend", conn) != true {
		return
	}

	////////////////////////////////////////////////////////////
	// initialize messages we'll be sending/receiving with frontend
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = frontendDataStruct{}
		inMsgReference  = frontendDataStruct{}
		outMsgTemp      = frontendDataStruct{}
		outMsgReference = frontendDataStruct{}
	)

	periodicUpdate := func() bool {

		inMsgTemp = frontendDataStruct{}
		outMsgTemp = frontendDataStruct{}

		////////////////////////////////////////////////////////////
		// put together outgoing message
		////////////////////////////////////////////////////////////

		GLOBAL.frontendDataMutex.RLock()

		foundMatch := false
		myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
		for i := 0; i < len(GLOBAL.frontendData.OtherFrontends); i++ {
			if GLOBAL.frontendData.OtherFrontends[i] == myIPaddr {
				foundMatch = true
				break
			}
		}
		if foundMatch == false {
			outMsgTemp.OtherFrontends = append(GLOBAL.frontendData.OtherFrontends, myIPaddr)
		}

		for i := 0; i < len(GLOBAL.frontendData.ListOfGalaxies); i++ {
			outMsgTemp.ListOfGalaxies[i].GalaxyName = GLOBAL.frontendData.ListOfGalaxies[i].GalaxyName
			outMsgTemp.ListOfGalaxies[i].GlobalX = GLOBAL.frontendData.ListOfGalaxies[i].GlobalX
			outMsgTemp.ListOfGalaxies[i].GlobalY = GLOBAL.frontendData.ListOfGalaxies[i].GlobalY
		}

		outMsgTemp.ListOfGalaxies = GLOBAL.frontendData.ListOfGalaxies
		outMsgTemp.OtherFrontends = GLOBAL.frontendData.OtherFrontends
		outMsgTemp.OtherFrontends = appendStringIfMissing(outMsgTemp.OtherFrontends, myIPaddr)

		GLOBAL.frontendDataMutex.RUnlock()

		////////////////////////////////////////////////////////////
		// defining constants to satisfy my linter
		////////////////////////////////////////////////////////////

		const (
			diediedie      = "diediedie"
			sameaslasttime = "same_as_last_time"
		)

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			outMsgTemp = frontendDataStruct{Message: sameaslasttime}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err = enc.Encode(outMsgTemp)
		if checkError("(periodicUpdateFrontend) error encoding outgoing message", err) {
			return false
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// wait for message from Frontend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError("(periodicUpdateFrontend) error receiving message:", err) {
			return false
		}
		if msgType != websocket.BinaryMessage {
			logInfo("(periodicUpdateFrontend) messageType is invalid")
			return false
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError("(periodicUpdateFrontend) error decoding incoming message", err) {
			return false
		}

		////////////////////////////////////////////////////////////
		// take a peek at the incoming message
		////////////////////////////////////////////////////////////

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == diediedie {
			// TODO // not implemented
		}

		////////////////////////////////////////////////////////////
		// making local copy of data structure to reference without contention
		////////////////////////////////////////////////////////////

		GLOBAL.frontendDataMutex.RLock()
		copyOfFrontendSlice := make([]string, len(GLOBAL.frontendData.OtherFrontends))
		copy(copyOfFrontendSlice, GLOBAL.frontendData.OtherFrontends)
		copyOfGalaxySlice := make([]galInfo, len(GLOBAL.frontendData.ListOfGalaxies))
		copy(copyOfGalaxySlice, GLOBAL.frontendData.ListOfGalaxies)
		GLOBAL.frontendDataMutex.RUnlock()

		////////////////////////////////////////////////////////////
		// append incoming frontend slice to local slice
		////////////////////////////////////////////////////////////

		newInfoFound := false
		for i := 0; i < len(inMsgTemp.OtherFrontends); i++ {
			if newInfoFound == true {
				break
			}
			remoteElem := inMsgTemp.OtherFrontends[i]
			for x := 0; x < len(copyOfFrontendSlice); x++ {
				localElem := copyOfFrontendSlice[x]
				if remoteElem == localElem {
					break
				}
				if x >= len(copyOfFrontendSlice)-1 {
					newInfoFound = true
				}
			}
		}

		if newInfoFound == true {
			GLOBAL.frontendDataMutex.Lock()
			for i := 0; i < len(GLOBAL.frontendData.OtherFrontends); i++ {
				GLOBAL.frontendData.OtherFrontends = appendStringIfMissing(GLOBAL.frontendData.OtherFrontends,
					GLOBAL.frontendData.OtherFrontends[i])
			}
			GLOBAL.frontendDataMutex.Unlock()
		}

		////////////////////////////////////////////////////////////
		// sort the galaxies by galaxyName
		////////////////////////////////////////////////////////////

		sort.Sort(ByGalaxyName(copyOfGalaxySlice))
		sort.Sort(ByGalaxyName(inMsgTemp.ListOfGalaxies))

		if reflect.DeepEqual(copyOfGalaxySlice, inMsgTemp.ListOfGalaxies) {
			logDebug("(periodicUpdateFrontend) localSlice of galaxies and incomingSlice are both properly sorted")
		} else {
			logDebug("(periodicUpdateFrontend) localSlice of galaxies needed to be sorted")
			GLOBAL.frontendDataMutex.Lock()
			sort.Sort(ByGalaxyName(GLOBAL.frontendData.ListOfGalaxies))
			GLOBAL.frontendDataMutex.Unlock()
		}

		////////////////////////////////////////////////////////////
		// check if galaxy exists and if it doesn't - add it to the slice
		////////////////////////////////////////////////////////////

		for i := 0; i < len(inMsgTemp.ListOfGalaxies); i++ {
			remoteElem := inMsgTemp.ListOfGalaxies[i].GalaxyName
			for x := 0; x < len(copyOfGalaxySlice); x++ {
				localElem := copyOfGalaxySlice[x].GalaxyName
				if remoteElem == localElem {
					break
				}
				if x >= len(copyOfGalaxySlice)-1 {
					GLOBAL.frontendDataMutex.Lock()
					GLOBAL.frontendData.ListOfGalaxies = append(GLOBAL.frontendData.ListOfGalaxies,
						inMsgTemp.ListOfGalaxies[i])
					GLOBAL.frontendDataMutex.Unlock()
				}
			}
		}

		////////////////////////////////////////////////////////////
		// append incoming galaxy specific slices to local slices
		////////////////////////////////////////////////////////////

		//	Simplified Example/Test:
		//	https://play.golang.org/p/ovy37tY29R

		for i := 0; i < len(inMsgTemp.ListOfGalaxies); i++ {
			thisGalaxysNewClientIDs := []string{}

			for x := 0; x < len(inMsgTemp.ListOfGalaxies[i].ClientIDs); x++ {
				remoteElem := inMsgTemp.ListOfGalaxies[i].ClientIDs[x]
				//fmt.Printf("i: %d \t remoteElem: %s \n", i, remoteElem)

				for y := 0; y < len(copyOfGalaxySlice); y++ {
					for z := 0; z < len(copyOfGalaxySlice[y].ClientIDs); z++ {
						localElem := copyOfGalaxySlice[y].ClientIDs[z]
						//fmt.Printf("i: %d \t localElem: %s \n", i, localElem)

						if remoteElem == localElem {
							break
						}

						if z >= len(copyOfGalaxySlice[y].ClientIDs)-1 {
							thisGalaxysNewClientIDs = append(thisGalaxysNewClientIDs, remoteElem)
						}
					}
				}
			}
			if len(thisGalaxysNewClientIDs) != 0 {
				GLOBAL.frontendDataMutex.Lock()
				existingData := &GLOBAL.frontendData.ListOfGalaxies[i].ClientIDs
				*existingData = append(*existingData, thisGalaxysNewClientIDs...)
				//fmt.Println("newFrontendData:", frontendData)
				GLOBAL.frontendDataMutex.Unlock()
			}
		}

		for i := 0; i < len(inMsgTemp.ListOfGalaxies); i++ {

			thisGalaxysNewBackends := []string{}
			for x := 0; x < len(inMsgTemp.ListOfGalaxies[i].ClusterOfBackends); x++ {

				remoteElem := inMsgTemp.ListOfGalaxies[i].ClusterOfBackends[x]
				//fmt.Printf("i: %d \t remoteElem: %s \n", i, remoteElem)
				for y := 0; y < len(copyOfGalaxySlice); y++ {

					for z := 0; z < len(copyOfGalaxySlice[y].ClusterOfBackends); z++ {
						localElem := copyOfGalaxySlice[y].ClusterOfBackends[z]
						//fmt.Printf("i: %d \t localElem: %s \n", i, localElem)

						if remoteElem == localElem {
							break
						}

						if z >= len(copyOfGalaxySlice[y].ClusterOfBackends)-1 {
							thisGalaxysNewBackends = append(thisGalaxysNewBackends, remoteElem)
						}
					}
				}
			}
			if len(thisGalaxysNewBackends) != 0 {
				GLOBAL.frontendDataMutex.Lock()
				existingData := &GLOBAL.frontendData.ListOfGalaxies[i].ClusterOfBackends
				*existingData = append(*existingData, thisGalaxysNewBackends...)
				//fmt.Println("newFrontendData:", frontendData)
				GLOBAL.frontendDataMutex.Unlock()
			}
		}

		////////////////////////////////////////////////////////////
		// store messsages for future reference
		////////////////////////////////////////////////////////////

		inMsgReference = inMsgTemp

		if outMsgTemp.Message != sameaslasttime {
			outMsgReference = outMsgTemp
		}

		return true
	}

	////////////////////////////////////////////////////////////
	// loop periodicUpdate and fallback to main if failure occurs
	////////////////////////////////////////////////////////////

	for {
		if periodicUpdate() == true {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	// fallback to main
	time.Sleep(1 * time.Second)
	return
}
