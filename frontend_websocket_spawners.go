package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

func spawnFrontendToFrontend(host string) {
	fName := "(handshakeFrontend)"

	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError(err, fName, "unable to open websocket with frontend loadbalancer") {
		return
	}
	defer conn.Close()

	logDebug(fName, "opening connection:", conn.RemoteAddr().String())
	defer logDebug(fName, "closing connection:", conn.RemoteAddr().String())

	if kickoffChallengeResponse("handshakeFrontend", conn) != true {
		return
	}

	////////////////////////////////////////////////////////////
	// initialize messages we'll be sending/receiving with frontend
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = frontendToFrontendMessage{}
		inMsgReference  = frontendToFrontendMessage{}
		outMsgTemp      = frontendToFrontendMessage{}
		outMsgReference = frontendToFrontendMessage{}
	)

	periodicUpdate := func() bool {

		inMsgTemp = frontendToFrontendMessage{}
		outMsgTemp = frontendToFrontendMessage{}

		////////////////////////////////////////////////////////////
		// put together outgoing message
		////////////////////////////////////////////////////////////

		GLOBAL.frontendData.RLock()
		for k, v := range GLOBAL.frontendData.GalGlobalX {
			outMsgTemp.GalGlobalX[k] = v
		}
		for k, v := range GLOBAL.frontendData.GalGlobalY {
			outMsgTemp.GalGlobalY[k] = v
		}
		for k, v := range GLOBAL.frontendData.GalGalaxyType {
			outMsgTemp.GalGalaxyType[k] = v
		}
		for k, vals := range GLOBAL.frontendData.GalClientIDs {
			newSlice := []string{}
			copy(newSlice, vals)
			outMsgTemp.GalClientIDs[k] = newSlice
		}
		for k, vals := range GLOBAL.frontendData.GalBackends {
			newSlice := []string{}
			copy(newSlice, vals)
			outMsgTemp.GalBackends[k] = newSlice
		}
		copy(outMsgTemp.OtherFrontends, GLOBAL.frontendData.OtherFrontends)
		GLOBAL.frontendData.RUnlock()

		golf := func() bool {
			myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
			for _, val := range inMsgTemp.OtherFrontends {
				if val == myIPaddr {
					return true
				}
			}
			return false
		}()
		if !golf {
			myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
			GLOBAL.frontendData.Lock()
			GLOBAL.frontendData.OtherFrontends = append(GLOBAL.frontendData.OtherFrontends, myIPaddr)
			GLOBAL.frontendData.Unlock()
		}

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

		logDebug(fName, fmt.Sprint("outMsgTemp=\t", outMsgTemp))
		logDebug(fName, fmt.Sprint("outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			outMsgTemp = frontendToFrontendMessage{
				Message: sameaslasttime,
			}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err = enc.Encode(outMsgTemp)
		if checkError(err, fName, "error encoding outgoing message") {
			return false
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// wait for message from Frontend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError(err, fName, "error receiving message") {
			return false
		}
		if msgType != websocket.BinaryMessage {
			logInfo(fName, "websocket messageType is not BinaryMessage")
			return false
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError(err, fName, "error decoding incoming message") {
			return false
		}

		////////////////////////////////////////////////////////////
		// take a peek at the incoming message
		////////////////////////////////////////////////////////////

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fName, fmt.Sprint("inMsgTemp=\t\t", inMsgTemp))
		logDebug(fName, fmt.Sprint("inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == diediedie {
			// TODO // not implemented
		}

		////////////////////////////////////////////////////////////
		// append incoming frontend slice to local slice
		////////////////////////////////////////////////////////////

		GLOBAL.frontendData.RLock()
		alpha := reflect.DeepEqual(inMsgTemp.GalGlobalX, GLOBAL.frontendData.GalGlobalX)
		beta := reflect.DeepEqual(inMsgTemp.GalGlobalY, GLOBAL.frontendData.GalGlobalY)
		charlie := reflect.DeepEqual(inMsgTemp.GalGalaxyType, GLOBAL.frontendData.GalGalaxyType)
		delta := reflect.DeepEqual(inMsgTemp.GalClientIDs, GLOBAL.frontendData.GalClientIDs)
		echo := reflect.DeepEqual(inMsgTemp.GalBackends, GLOBAL.frontendData.GalBackends)
		foxtrot := reflect.DeepEqual(inMsgTemp.OtherFrontends, GLOBAL.frontendData.OtherFrontends)
		golf = func() bool {
			myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
			for _, val := range inMsgTemp.OtherFrontends {
				if val == myIPaddr {
					return true
				}
			}
			return false
		}()
		GLOBAL.frontendData.RUnlock()

		if !alpha || !beta || !charlie || !delta || !echo || !foxtrot || !golf {
			GLOBAL.frontendData.Lock()
			if !alpha {
				for galName, value := range inMsgTemp.GalGlobalX {
					_, ok := GLOBAL.frontendData.GalGlobalX[galName]
					if !ok {
						GLOBAL.frontendData.GalGlobalX[galName] = value
					}
				}
			}
			if !beta {
				for galName, value := range inMsgTemp.GalGlobalY {
					_, ok := GLOBAL.frontendData.GalGlobalY[galName]
					if !ok {
						GLOBAL.frontendData.GalGlobalY[galName] = value
					}
				}
			}
			if !charlie {
				for galName, value := range inMsgTemp.GalGalaxyType {
					_, ok := GLOBAL.frontendData.GalGalaxyType[galName]
					if !ok {
						GLOBAL.frontendData.GalGalaxyType[galName] = value
					}
				}
			}
			if !delta {
				for galName, remoteUUIDs := range inMsgTemp.GalClientIDs {
					_, ok := GLOBAL.frontendData.GalClientIDs[galName]
					if !ok {
						GLOBAL.frontendData.GalClientIDs[galName] = remoteUUIDs
					} else {
						for _, ID := range remoteUUIDs {
							appendStringIfMissing(GLOBAL.frontendData.GalClientIDs[galName], ID)
						}
					}
				}
			}
			if !echo {
				for galName, remoteBackends := range inMsgTemp.GalBackends {
					_, ok := GLOBAL.frontendData.GalBackends[galName]
					if !ok {
						GLOBAL.frontendData.GalBackends[galName] = remoteBackends
					} else {
						for _, backend := range remoteBackends {
							appendStringIfMissing(GLOBAL.frontendData.GalBackends[galName], backend)
						}
					}
				}
			}
			if !foxtrot {
				for _, remoteFrontend := range inMsgTemp.OtherFrontends {
					appendStringIfMissing(GLOBAL.frontendData.OtherFrontends, remoteFrontend)
				}
			}
			if !golf {
				myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
				GLOBAL.frontendData.OtherFrontends = append(GLOBAL.frontendData.OtherFrontends, myIPaddr)
			}
			GLOBAL.frontendData.Unlock()
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
