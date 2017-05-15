package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"sort"
	"time"

	cryptorand "crypto/rand"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v1"
)

func handleClientToFrontend(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	if checkFirewall("loadbalanceClients", client, "challenge") != true {
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
	if checkError("(loadbalanceClients) upgrade error:", err) {
		return
	}
	logDebug("(loadbalanceClients) opening socket:", string(r.RemoteAddr))

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	clientSessionCountMutex.Lock()
	_, ok := clientSessionCount[client]
	if ok {
		clientSessionCount[client]++
	} else {
		clientSessionCount[client] = 1
	}
	clientSessionCountMutex.Unlock()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
	}()

	////////////////////////////////////////////////////////////
	// receive JSON message from client
	////////////////////////////////////////////////////////////

	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if checkError("(loadbalanceClients) error receiving message:", err) {
		return
	}
	logDebug("(loadbalanceClients) received the following message\n", fmt.Sprintf("%s", inMsg))

	////////////////////////////////////////////////////////////
	// send id_token to google oauth service
	////////////////////////////////////////////////////////////

	successful := false
	var oauth2Service *oauth2.Service // will be used outside of 'for' loop
	for i := 0; i < constGoogleAuth; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if checkError("(loadbalanceClients) error contacting google:", err) {
			continue
		} else {
			successful = true
			break
		}
	}
	if successful == false {
		currentTime := time.Now().Unix()
		clientBlacklist[client].Lock()
		clientBlacklist[client].failedGoogleAuthCount += constGoogleAuth
		clientBlacklist[client].failedGoogleAuthTime = currentTime
		clientBlacklist[client].Unlock()
		return
	}

	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.IDToken)
	tokenInfo, err := tokenInfoCall.Do()
	if checkError("(loadbalanceClients) error parsing incoming google response:", err) {
		conn.Close()
		return
	}

	rawJSON, err := tokenInfo.MarshalJSON()
	if checkError("(loadbalanceClients) error MarshalJSON'ing google response:", err) {
		conn.Close()
		return
	}

	if debugModeBool == true {
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		checkError("(loadbalanceClients) error jsonIndent'ing google response:", err)
		logDebug("(loadbalanceClients) received the following message from google:\n",
			string(prettyJSON.Bytes()))
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
	// close websocket with completion
	////////////////////////////////////////////////////////////

	logDebug("(loadbalanceClients) closing socket:", string(r.RemoteAddr))
}

func handleBackendToFrontend(w http.ResponseWriter, r *http.Request) {

	// if client is blocked - backout of this http connection with 'return'
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	thisCheckPassed := checkFirewall("handshakeBackend", client, "challenge")
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
	if checkError("(handshakeBackend) upgrade error:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// defer statements
	////////////////////////////////////////////////////////////

	defer conn.Close()

	clientSessionCountMutex.Lock()
	_, ok := clientSessionCount[client]
	if ok {
		clientSessionCount[client]++
	} else {
		clientSessionCount[client] = 1
		logDebug("(handshakeBackend) setting 'clientSessionCount' to ONE",
			"for this client:", client)
	}
	clientSessionCountMutex.Unlock()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
	}()

	logDebug("(handshakeBackend) opening socket:", string(r.RemoteAddr))
	defer logDebug("(handshakeBackend) closing connection:", client)

	////////////////////////////////////////////////////////////
	// do challenge-response handshake (defined in Main.go)
	////////////////////////////////////////////////////////////

	successful := handleChallengeResponse("handshakeBackend", conn, client)
	if successful == false {
		return
	}

	////////////////////////////////////////////////////////////
	// define data structures for upcoming back/forth communication
	////////////////////////////////////////////////////////////

	type incomingMessageBackend struct {
		ServerAddress     string   `json:"ServerAddress"`
		GalaxyName        string   `json:"GalaxyName"`
		ActiveClientCount int      `json:"ActiveClientCount"`
		TotalClientCount  int      `json:"TotalClientCount"`
		OtherFrontends    []string `json:"OtherFrontends"`
		OtherBackends     []string `json:"OtherBackends"`
		Message           string   `json:"Message"`
		AnotherMessage    string   `json:"AnotherMessage"`
	}

	type outgoingMessageBackend struct {
		GalaxyName     string   `json:"GalaxyName"`
		GlobalX        int64    `json:"GlobalX"`
		GlobalY        int64    `json:"GlobalY"`
		OtherFrontends []string `json:"OtherFrontends"`
		OtherBackends  []string `json:"OtherBackends"`
		Message        string   `json:"Message"`
		AnotherMessage string   `json:"AnotherMessage"`
	}

	var (
		inMsgTemp       = incomingMessageBackend{}
		inMsgReference  = incomingMessageBackend{}
		outMsgTemp      = outgoingMessageBackend{}
		outMsgReference = outgoingMessageBackend{}
	)

	for {

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = incomingMessageBackend{}
		outMsgTemp = outgoingMessageBackend{}

		////////////////////////////////////////////////////////////
		// wait for message from Backend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError("(periodicUpdateBackend) error receiving message:", err) {
			return
		}
		if msgType != websocket.BinaryMessage {
			logInfo("(periodicUpdateBackend) messageType is invalid")
			return
		}
		network := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(network)
		err = dec.Decode(&inMsgTemp)
		if checkError("(periodicUpdateBackend) error decoding incoming message", err) {
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
			diediedie      = "diediedie"
			sameaslasttime = "same_as_last_time"
			yourgalaxyname = "your_galaxy_name"
		)

		////////////////////////////////////////////////////////////
		// look at the incoming message and act on it
		////////////////////////////////////////////////////////////

		galName := outMsgReference.GalaxyName
		logDebug(fmt.Sprint("(periodicUpdateBackend) galName=\t\t", galName))
		// TODO // need to reference galaxyInfo instead

		if inMsgTemp.Message == sameaslasttime {
			inMsgTemp = inMsgReference
		}

		logDebug(fmt.Sprint("(periodicUpdateBackend) inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateBackend) inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == sameaslasttime && galName != "" {
			if reflect.DeepEqual(inMsgTemp, inMsgReference) {
				logInfo("(periodicUpdateBackend) received duplicate message")
			}
		}

		////////////////////////////////////////////////////////////
		// handle the backend requesting for newGalaxy generation
		////////////////////////////////////////////////////////////

		if inMsgTemp.Message == newgalaxy {
			logInfo("(periodicUpdateBackend) received 'new_galaxy' from backend")
			myGalName, glblX, glblY := generateGlobalGalaxyInfo(client)

			galName = myGalName
			outMsgTemp.GalaxyName = galName
			outMsgTemp.Message = yourgalaxyname

			frontendDataMutex.RLock()
			galIndex := frontendData.GetGalIndexOf(galName)
			frontendDataMutex.RUnlock()
			if galIndex == -1 {
				frontendDataMutex.Lock()
				newObjectForSlice := galInfo{}
				newObjectForSlice.GalaxyName = galName
				newObjectForSlice.GlobalX = glblX
				newObjectForSlice.GlobalY = glblY
				newObjectForSlice.ClientIDs = []string{}
				newObjectForSlice.ClusterOfBackends = []string{client}
				// TODO // need to remove backend from galaxyInfo.Backends on disconnect
				frontendData.ListOfGalaxies = append(frontendData.ListOfGalaxies, newObjectForSlice)
				frontendDataMutex.Unlock()
			} else {
				logInfo("(periodicUpdateBackend) galaxy that we JUST generated already exists (fatalerror)")
				return
			}
			logInfo(fmt.Sprintf("(periodicUpdateBackend) generated the following galInfo:"+
				"\n\t\tgalName=%s\n\t\tglblX=%d\n\t\tglblY=%d", galName, glblX, glblY))
		}

		if inMsgTemp.Message == diediedie {
			logInfo("(periodicUpdateBackend) received 'diediedie' from backend *ignoring*")
			// TODO // add sanity logic to 'diediedie' message
		}

		////////////////////////////////////////////////////////////
		// finish putting together outgoing message
		////////////////////////////////////////////////////////////

		if galName == "" && inMsgTemp.GalaxyName != "" {
			logInfo(fmt.Sprintf("(periodicUpdateBackend) something is horribly wrong..."+
				" backend %s is asserting unknown galName", client))
			return
		}

		if galName == "" && inMsgTemp.Message != newgalaxy {
			logInfo("(periodicUpdateBackend) invalid incoming request: empty galName and invalid message.newGalaxy")
			return
		}

		if galName == "" {
			logInfo("(periodicUpdateBackend) something is terribly wrong... galaxy was not created/initialized")
			return
		}

		outMsgTemp.GalaxyName = galName

		frontendDataMutex.RLock()
		galIndex := frontendData.GetGalIndexOf(galName)
		outMsgTemp.GlobalX = frontendData.ListOfGalaxies[galIndex].GlobalX
		outMsgTemp.GlobalY = frontendData.ListOfGalaxies[galIndex].GlobalY
		outMsgTemp.OtherFrontends = frontendData.OtherFrontends
		outMsgTemp.OtherBackends = frontendData.ListOfGalaxies[galIndex].ClusterOfBackends
		frontendDataMutex.RUnlock()

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint("(periodicUpdateBackend) outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateBackend) outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			outMsgTemp = outgoingMessageBackend{Message: sameaslasttime}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err = enc.Encode(outMsgTemp)
		if checkError("(periodicUpdateBackend) error encoding outgoing message", err) {
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
	if checkError("(handshakeFrontend) upgrade error:", err) {
		return
	}
	defer conn.Close()

	// log client IP address
	logInfo("(handshakeFrontend) opening socket:", string(r.RemoteAddr))
	defer logInfo("(handshakeFrontend) closing connection:", client)

	////////////////////////////////////////////////////////////
	// modify sessionCount accordingly
	////////////////////////////////////////////////////////////

	clientSessionCountMutex.Lock()
	_, ok := clientSessionCount[client]
	if ok {
		clientSessionCount[client]++
	} else {
		clientSessionCount[client] = 1
		logDebug("(handshakeFrontend) setting 'clientSessionCount' to ONE",
			"for this client:", client)
	}
	clientSessionCountMutex.Unlock()

	defer func() {
		clientSessionCountMutex.Lock()
		clientSessionCount[client]--
		clientSessionCountMutex.Unlock()
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
		inMsgTemp       = FrontendDataStruct{}
		inMsgReference  = FrontendDataStruct{}
		outMsgTemp      = FrontendDataStruct{}
		outMsgReference = FrontendDataStruct{}
	)

	////////////////////////////////////////////////////////////
	// start communication with other frontend
	////////////////////////////////////////////////////////////

	for {

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = FrontendDataStruct{}
		outMsgTemp = FrontendDataStruct{}

		////////////////////////////////////////////////////////////
		// wait for message from Frontend
		////////////////////////////////////////////////////////////

		msgType, newData, err := conn.ReadMessage()
		if checkError("(periodicUpdateFrontend) error receiving message:", err) {
			return
		}
		if msgType != websocket.BinaryMessage {
			logInfo("(periodicUpdateFrontend) messageType is invalid")
			return
		}
		networkIn := bytes.NewBuffer(newData)
		dec := gob.NewDecoder(networkIn)
		err = dec.Decode(&inMsgTemp)
		if checkError("(periodicUpdateFrontend) error decoding incoming message", err) {
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

		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgTemp=\t\t", inMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) inMsgReference=\t", inMsgReference))

		if inMsgTemp.Message == diediedie {
			// TODO // not implemented
		}

		////////////////////////////////////////////////////////////
		// append incoming frontend slice to local slice
		////////////////////////////////////////////////////////////

		frontendDataMutex.RLock()
		copyOfFrontendSlice := make([]string, len(frontendData.OtherFrontends))
		copy(copyOfFrontendSlice, frontendData.OtherFrontends)
		copyOfGalaxySlice := make([]galInfo, len(frontendData.ListOfGalaxies))
		copy(copyOfGalaxySlice, frontendData.ListOfGalaxies)
		frontendDataMutex.RUnlock()

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
			frontendDataMutex.Lock()
			for i := 0; i < len(frontendData.OtherFrontends); i++ {
				frontendData.OtherFrontends = appendStringIfMissing(frontendData.OtherFrontends,
					frontendData.OtherFrontends[i])
			}
			frontendDataMutex.Unlock()
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
			frontendDataMutex.Lock()
			sort.Sort(ByGalaxyName(frontendData.ListOfGalaxies))
			frontendDataMutex.Unlock()
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
					frontendDataMutex.Lock()
					frontendData.ListOfGalaxies = append(frontendData.ListOfGalaxies, inMsgTemp.ListOfGalaxies[i])
					frontendDataMutex.Unlock()
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
				frontendDataMutex.Lock()
				existingData := &frontendData.ListOfGalaxies[i].ClientIDs
				*existingData = append(*existingData, thisGalaxysNewClientIDs...)
				//fmt.Println("newFrontendData:", frontendData)
				frontendDataMutex.Unlock()
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
				frontendDataMutex.Lock()
				existingData := &frontendData.ListOfGalaxies[i].ClusterOfBackends
				*existingData = append(*existingData, thisGalaxysNewBackends...)
				//fmt.Println("newFrontendData:", frontendData)
				frontendDataMutex.Unlock()
			}
		}

		////////////////////////////////////////////////////////////
		// finish putting together outgoing message
		////////////////////////////////////////////////////////////

		frontendDataMutex.RLock()

		foundMatch := false
		myIPaddr, _, _ := net.SplitHostPort(conn.LocalAddr().String())
		for i := 0; i < len(frontendData.OtherFrontends); i++ {
			if frontendData.OtherFrontends[i] == myIPaddr {
				foundMatch = true
				break
			}
		}
		if foundMatch == false {
			outMsgTemp.OtherFrontends = append(frontendData.OtherFrontends, myIPaddr)
		}

		for i := 0; i < len(copyOfGalaxySlice); i++ {
			outMsgTemp.ListOfGalaxies[i].GalaxyName = frontendData.ListOfGalaxies[i].GalaxyName
			outMsgTemp.ListOfGalaxies[i].GlobalX = frontendData.ListOfGalaxies[i].GlobalX
			outMsgTemp.ListOfGalaxies[i].GlobalY = frontendData.ListOfGalaxies[i].GlobalY
		}

		frontendDataMutex.RUnlock()

		////////////////////////////////////////////////////////////
		// send outgoingMessage
		////////////////////////////////////////////////////////////

		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgTemp=\t", outMsgTemp))
		logDebug(fmt.Sprint("(periodicUpdateFrontend) outMsgReference=\t", outMsgReference))

		if reflect.DeepEqual(outMsgTemp, outMsgReference) {
			outMsgTemp = FrontendDataStruct{Message: sameaslasttime}
		}

		var networkOut bytes.Buffer
		enc := gob.NewEncoder(&networkOut)
		err = enc.Encode(outMsgTemp)
		if checkError("(periodicUpdateFrontend) error encoding outgoing message", err) {
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
