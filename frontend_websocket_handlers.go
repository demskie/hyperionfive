package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"reflect"

	cryptorand "crypto/rand"

	"github.com/gorilla/websocket"
)

func handleClientToFrontend(w http.ResponseWriter, r *http.Request) {
	fName := "(clientToFrontend)"

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	remoteIPAddr, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	if checkFirewall(fName, remoteIPAddr, "challenge") != true {
		return
	}

	////////////////////////////////////////////////////////////
	// upgrade websocket
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName) {
		return
	}
	logDebug(fName, "opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[remoteIPAddr]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive and process google authentication
	////////////////////////////////////////////////////////////

	userid := handleGoogleAuth(fName, conn, remoteIPAddr)
	if userid == "" {
		return
	}

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
	// TODO: RESPONSE TO CLIENT NOT YET IMPLEMENTED
	////////////////////////////////////////////////////////////

	logDebug(fName, "closing socket:", string(r.RemoteAddr))
}

func handleBackendToFrontend(w http.ResponseWriter, r *http.Request) {
	fName := "(backendToFrontend)"

	// if client is blocked - backout of this http connection with 'return'
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	thisCheckPassed := checkFirewall(fName, client, "challenge")
	if thisCheckPassed != true {
		return
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName) {
		return
	}

	////////////////////////////////////////////////////////////
	// defer statements
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[client]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[client]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[client] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[client]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	logDebug(fName, "opening socket:", string(r.RemoteAddr))
	defer logDebug(fName, "closing connection:", client)

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse(fName, conn, client)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = backendToFrontendMessage{}
		inMsgReference  = backendToFrontendMessage{}
		outMsgTemp      = frontendToBackendMessage{}
		outMsgReference = frontendToBackendMessage{}
	)

	for {

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = backendToFrontendMessage{}
		outMsgTemp = frontendToBackendMessage{}

		////////////////////////////////////////////////////////////
		// wait for message from Backend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError(err, fName, "error receiving message") {
			return
		}
		if msgType != websocket.BinaryMessage {
			logInfo(fName, "messageType is invalid")
			return
		}
		network := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(network)
		err = dec.Decode(&inMsgTemp)
		if checkError(err, fName, "error decoding incoming message") {
			return
		}

		////////////////////////////////////////////////////////////
		// define function used to generate global Galaxy Name and Coordinates
		////////////////////////////////////////////////////////////

		generateGlobalGalaxyInfo := func(clientIP string) (galName string, glblX int64, glblY int64) {

			lotsOfNames := []string{"Aegis", "Aeon", "Aeris", "Babylon", "Aeternitas", "Aether", "Alliance", "Alpha", "Amazone", "Ancestor", "Anemone", "Angel", "Anomaly", "Apollo", "Arcadia", "Arcadis", "Arch", "Architect", "Ark", "Artemis", "Asphodel", "Asteria", "Astraeus", "Athena", "Atlas", "Atmos", "Aura", "Aurora", "Awe", "Azura", "Azure", "Baldur", "Beacon", "Blue Moon", "Borealis", "Burrow", "Caelestis", "Canaan", "Century", "Chrono", "Chronos", "Crescent", "Curator", "Curiosity", "Data", "Dawn", "Daydream", "Demeter", "Dogma", "Dream", "Dune", "Ecstacys", "Eir", "Elyse", "Elysium", "Empyrea", "Ender", "Enigma", "Eos", "Epiphany", "Epitome", "Erebus", "Escort", "Eternis", "Eternity", "Exposure", "Fable", "Father", "Fauna", "Felicity", "Flora", "Fortuna", "Frontier", "Gaia", "Galaxy", "Genesis", "Genius", "Glory", "Guardian", "Halo", "Heirloom", "Helios", "Hemera", "Hera", "Heritage", "Hermes", "Horus", "Hymn", "Hyperion", "Hypnos", "Ignis", "Illume", "Inception", "Infinity", "Isis", "Janus", "Juno", "Legacy", "Liberty", "Lore", "Lucent", "Lumina", "Luminous", "Luna", "Lunis", "Magni", "Mammoth", "Mani", "Marvel", "Memento", "Minerva", "Miracle", "Mother", "Muse", "Mystery", "Mythos", "Nebula", "Nemesis", "Nemo", "Neo", "Nero", "Nimbus", "Nott", "Nova", "Novis", "Nox", "Nyx", "Odyssey", "Olympus", "Omega", "Oracle", "Orbital", "Origin", "Orphan", "Osiris", "Outlander", "Parable", "Paradox", "Paragon", "Pedigree", "Phantasm", "Phantom", "Phenomenon", "Phoenix", "Pilgrim", "Pioneer", "Prism", "Prodigy", "Prometheus", "Prophecy", "Proto", "Radiance", "Rebus", "Relic", "Revelation", "Reverie", "Rogue", "Rune", "Saga", "Sancus", "Scout", "Selene", "Serenity", "Settler", "Shangris", "Shepherd", "Shu", "Sol", "Solas", "Spectacle", "Specter", "Spectrum", "Spire", "Symbolica", "Tartarus", "Terminus", "Terra", "Terran", "Terraria", "Themis", "Tiberius", "Titan", "Titanus", "Torus", "Tranquility", "Trivia", "Utopis", "Valhalla", "Vanguard", "Vanquish", "Vesta", "Vestige", "Victoria", "Virtue", "Visage", "Voyage", "Vulcan", "Warden", "Yggdrasil", "Zeus", "Zion"}

			seed := func() rand.Source {
				var randomness int64
				b := make([]byte, 8)
				cryptorand.Read(b)
				buf := bytes.NewBuffer(b)
				binary.Read(buf, binary.LittleEndian, &randomness)
				return rand.NewSource(randomness)
			}
			rnum := rand.New(seed())
			randomIndex := rnum.Intn(len(lotsOfNames) - 1)

			galName = lotsOfNames[randomIndex]

			for {
				////////////////////////////////////////////
				// generating a uniformly random int64 value
				////////////////////////////////////////////
				b := make([]byte, 8)
				cryptorand.Read(b)
				buf := bytes.NewBuffer(b)
				binary.Read(buf, binary.LittleEndian, &glblX)

				//////////////////////////////////////////////////
				// generating another uniformly random int64 value
				//////////////////////////////////////////////////
				cryptorand.Read(b)
				buf = bytes.NewBuffer(b)
				binary.Read(buf, binary.LittleEndian, &glblY)

				/////////////////////////////////////////////////
				// calculate hypotenuse and throw away any
				// coordinates that would fall outside the circle
				/////////////////////////////////////////////////
				A2 := float64(glblX * glblX)
				B2 := float64(glblY * glblY)
				C := math.Sqrt(A2 + B2)

				if C >= galaxyRadius {
					continue
				}
				break
			}
			return galName, glblX, glblY
		}

		// defining constants to satisfy my linter
		const (
			newgalaxy      = "new_galaxy"
			diediedie      = "die_die_die"
			sameaslasttime = "same_as_last_time"
			yourgalaxyname = "your_galaxy_name"
		)

		////////////////////////////////////////////////////////////
		// look at the incoming message and act on it
		////////////////////////////////////////////////////////////

		galName := outMsgReference.GalaxyName
		logDebug(fName, fmt.Sprint("galName=\t\t", galName))
		// TODO // need to reference galaxyInfo instead

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fName, fmt.Sprint("inMsgTemp=\t\t", inMsgTemp))
		logDebug(fName, fmt.Sprint("inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == sameaslasttime && galName != "" {
			if reflect.DeepEqual(inMsgTemp, inMsgReference) {
				logInfo(fName, "received duplicate message")
			}
		}

		////////////////////////////////////////////////////////////
		// handle the backend requesting for newGalaxy generation
		////////////////////////////////////////////////////////////

		if inMsgTemp.Message == newgalaxy {
			logInfo(fName, "received 'new_galaxy' from backend")
			myGalName, glblX, glblY := generateGlobalGalaxyInfo(client)

			galName = myGalName
			outMsgTemp.GalaxyName = galName
			outMsgTemp.Message = yourgalaxyname

			galaxyIsNotDefined := true
			GLOBAL.frontendData.RLock()
			for gn := range GLOBAL.frontendData.GalGalaxyType {
				if gn == galName {
					galaxyIsNotDefined = false
					break
				}
			}
			GLOBAL.frontendData.RLock()

			if galaxyIsNotDefined {
				GLOBAL.frontendData.Lock()
				GLOBAL.frontendData.GalGlobalX[galName] = glblX
				GLOBAL.frontendData.GalGlobalY[galName] = glblY
				GLOBAL.frontendData.GalGalaxyType[galName] = 99 // TODO: IMPLEMENT
				GLOBAL.frontendData.GalClientIDs[galName] = []string{}
				GLOBAL.frontendData.GalBackends[galName] = []string{client}
				GLOBAL.frontendData.Unlock()
			} else {
				logError(fName, "just tried to generate a galaxy that already exists!")
				return
			}
			logInfo(fName, fmt.Sprintf("generated the following galInfo:"+
				"\n\t\tgalName=%s\n\t\tglblX=%d\n\t\tglblY=%d", galName, glblX, glblY))
		}

		if inMsgTemp.Message == diediedie {
			logInfo(fName + "received 'diediedie' from backend *ignoring*")
			// TODO // add sanity logic to 'diediedie' message
		}

		////////////////////////////////////////////////////////////
		// finish putting together outgoing message
		////////////////////////////////////////////////////////////

		if galName == "" && inMsgTemp.GalaxyName != "" {
			logInfo(fName, fmt.Sprintf("something is horribly wrong..."+
				" backend %s is asserting unknown galName", client))
			return
		}

		if galName == "" && inMsgTemp.Message != newgalaxy {
			logInfo(fName, "invalid incoming request: empty galName and invalid message.newGalaxy")
			return
		}

		if galName == "" {
			logInfo(fName, "something is terribly wrong... galaxy was not created/initialized")
			return
		}

		outMsgTemp.GalaxyName = galName
		GLOBAL.frontendData.RLock()
		outMsgTemp.GlobalX = GLOBAL.frontendData.GalGlobalX[galName]
		outMsgTemp.GlobalY = GLOBAL.frontendData.GalGlobalY[galName]
		outMsgTemp.OtherFrontends = GLOBAL.frontendData.OtherFrontends
		outMsgTemp.OtherBackends = GLOBAL.frontendData.GalBackends[galName]
		GLOBAL.frontendData.RUnlock()

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fName, fmt.Sprint("outMsgTemp=\t", outMsgTemp))
		logDebug(fName, fmt.Sprint("outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			outMsgTemp = frontendToBackendMessage{
				Message: sameaslasttime,
			}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err = enc.Encode(outMsgTemp)
		if checkError(err, fName, "error encoding outgoing message") {
			return
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// store messsages for future reference
		////////////////////////////////////////////////////////////

		inMsgReference = inMsgTemp

		if outMsgTemp.Message != sameaslasttime {
			outMsgReference = outMsgTemp
		}
	}
}

func handleFrontendToFrontend(w http.ResponseWriter, r *http.Request) {
	fName := "(handshakeFrontend)"

	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	// if client is blocked - backout of this http connection with 'return'
	thisCheckPassed := checkFirewall("handshakeFrontend", client, "challenge")
	if thisCheckPassed != true {
		return
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError(err, fName) {
		return
	}

	// log client IP address
	logInfo(fName, "opening socket:", string(r.RemoteAddr))
	defer logInfo(fName, "closing connection:", client)

	////////////////////////////////////////////////////////////
	// modify sessionCount accordingly
	////////////////////////////////////////////////////////////

	defer conn.Close()

	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.TotalSessionCount[client]
	if ok {
		GLOBAL.clientBlacklist.TotalSessionCount[client]++
	} else {
		GLOBAL.clientBlacklist.TotalSessionCount[client] = 1
	}
	GLOBAL.clientBlacklist.Unlock()

	defer func() {
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.TotalSessionCount[client]--
		GLOBAL.clientBlacklist.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse("handshakeFrontend", conn, client)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = frontendToFrontendMessage{}
		inMsgReference  = frontendToFrontendMessage{}
		outMsgTemp      = frontendToFrontendMessage{}
		outMsgReference = frontendToFrontendMessage{}
	)

	////////////////////////////////////////////////////////////
	// start communication with other frontend
	////////////////////////////////////////////////////////////

	for {

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = frontendToFrontendMessage{}
		outMsgTemp = frontendToFrontendMessage{}

		////////////////////////////////////////////////////////////
		// wait for message from Frontend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError(err, fName) {
			return
		}
		if msgType != websocket.BinaryMessage {
			logInfo(fName, "messageType is invalid")
			return
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError(err, fName) {
			return
		}

		////////////////////////////////////////////////////////////
		// defining constants to satisfy my linter
		////////////////////////////////////////////////////////////

		const (
			diediedie      = "diediedie"
			sameaslasttime = "same_as_last_time"
		)

		////////////////////////////////////////////////////////////
		// look at the incoming message and act on it
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
		golf := func() bool {
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
		// finish putting together outgoing message
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
		if checkError(err, fName) {
			return
		}
		conn.WriteMessage(websocket.BinaryMessage, networkOut.Bytes())

		////////////////////////////////////////////////////////////
		// store messsages for future reference
		////////////////////////////////////////////////////////////

		inMsgReference = inMsgTemp

		if outMsgTemp.Message != sameaslasttime {
			outMsgReference = outMsgTemp
		}
	}
}
