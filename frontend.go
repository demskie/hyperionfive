package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v1"

	cryptorand "crypto/rand"
)

func startFrontendMode() {

	// initialize frontendData
	frontendData = FrontendDataStruct{}

	// setup httpMux to avoid leaking default handlers
	muxSecure := http.NewServeMux()

	// to enable perfect forward secrecy it's necessary to add 'Strict-Transport-Security'
	// and since we are not writing out own FileServer, we'll need to add Headers this way
	addSecurityHeaders := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.ServeHTTP(w, r)
		}
	}
	directory := http.Dir("D:\\OneDrive\\GoPrototypes\\src\\GOSERVER\\websiteroot\\")
	fileserverWithModifiedHeaders := addSecurityHeaders(http.FileServer(directory))

	// setup the fileserver handler now that the necessary security headers have been added
	muxSecure.Handle("/", fileserverWithModifiedHeaders)

	// setup the websocket handlers
	muxSecure.HandleFunc("/clientlb", frontendReceiveClientInfoRequest)
	muxSecure.HandleFunc("/backend", frontendReceiveBackendInfoRequest)
	muxSecure.HandleFunc("/frontend", frontendReceiveFrontendInfoRequest)

	// tying the following http parameters to the handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         ":443",
		Handler:      muxSecure,

		// defining various security parameters used to specify accepted ciphersuites
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},

		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// define waitgroup to be used to stop premature termination
	var lbWaitGroup sync.WaitGroup

	// serve HTTPS requests with config defined above
	lbWaitGroup.Add(1)
	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			waitForLogging.Wait()
			//os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// define nonsecure HTTP parameters
	srvInsecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			url := "https://" + req.Host + req.URL.String()
			http.Redirect(w, req, url, http.StatusMovedPermanently)
		}),
	}

	// serve HTTP requests by redirecting to HTTPS
	lbWaitGroup.Add(1)
	go func() {
		errServeInsecure := srvInsecure.ListenAndServe()
		if checkError("(ListenAndServeInsecure) Something went wrong:", errServeInsecure) {
			waitForLogging.Wait()
			//os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// use waitgroup to stop permature termination unless all
	// the above goroutines are finished executing
	lbWaitGroup.Wait()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func frontendReceiveClientInfoRequest(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	// parsing IP address of this new client
	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	clientBlacklistMutex.RLock()
	clientFirewallValues, exists := clientBlacklist[client]
	clientBlacklistMutex.RUnlock()
	if exists == true {
		clientFirewallValues.RLock()
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - clientFirewallValues.failedChallengeTime
		blacklistExpGrowth := int64((clientFirewallValues.failedChallengeCount) / constChallenge)
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(loadbalanceClients) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(loadbalanceClients) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedGoogleAuth",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedGoogleAuthCount))
			return
		}
		clientFirewallValues.RUnlock()
	} else {
		// intialize client with default values
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = &FirewallClientValues{}
		clientBlacklistMutex.Unlock()
	}

	////////////////////////////////////////////////////////////
	// begin processing incoming message
	////////////////////////////////////////////////////////////

	// defining websocket.Upgrader parameters
	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	// attempting to process the incoming client's request
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(loadbalanceClients) upgrade error:", err) {
		return
	}
	logDebug("(loadbalanceClients) opening socket:", string(r.RemoteAddr))

	// enable write compression

	////////////////////////////////////////////////////////////
	// DEFER STATEMENTS
	////////////////////////////////////////////////////////////

	defer conn.Close()

	go func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]++
		} else {
			clientSessionCount[client] = 1
		}
		clientSessionCountMutex.Unlock()
	}()

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
		clientFirewallValues.Lock()
		clientFirewallValues.failedGoogleAuthCount += constGoogleAuth
		clientFirewallValues.failedGoogleAuthTime = currentTime
		clientFirewallValues.Unlock()
		conn.Close()
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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func frontendReceiveBackendInfoRequest(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	// parsing IP address of this new client
	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	clientBlacklistMutex.RLock()
	clientFirewallValues, exists := clientBlacklist[client]
	clientBlacklistMutex.RUnlock()
	if exists == true {
		clientFirewallValues.RLock()
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - clientFirewallValues.failedChallengeTime
		blacklistExpGrowth := int64((clientFirewallValues.failedChallengeCount) / constChallenge)
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeBackend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeBackend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedGoogleAuth",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedGoogleAuthCount))
			return
		}
		clientFirewallValues.RUnlock()
	} else {
		// intialize client with default values
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = &FirewallClientValues{}
		clientBlacklistMutex.Unlock()
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
	defer conn.Close()

	// log client IP address
	logDebug("(handshakeBackend) opening socket:", string(r.RemoteAddr))
	defer logDebug("(handshakeBackend) closing connection:", client)

	////////////////////////////////////////////////////////////
	// modify sessionCount accordingly
	////////////////////////////////////////////////////////////

	go func() {
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
	}()

	defer func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]--
		} else {
			clientSessionCount[client] = 0
			logDebug("(handshakeBackend) setting 'clientSessionCount' to ZERO",
				"for this client:", client)
		}
		clientSessionCountMutex.Unlock()
	}()

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

	////////////////////////////////////////////////////////////
	// start communication with backend
	////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func frontendReceiveFrontendInfoRequest(w http.ResponseWriter, r *http.Request) {

	////////////////////////////////////////////////////////////
	// check that the host is not currently blacklisted
	////////////////////////////////////////////////////////////

	// parsing IP address of this new client
	client, _, _ := net.SplitHostPort(r.RemoteAddr)

	// checking blacklist and acting on it accordingly
	clientBlacklistMutex.RLock()
	clientFirewallValues, exists := clientBlacklist[client]
	clientBlacklistMutex.RUnlock()
	if exists == true {
		clientFirewallValues.RLock()
		currentTime := time.Now().Unix()
		timeSinceBlacklist := currentTime - clientFirewallValues.failedChallengeTime
		blacklistExpGrowth := int64((clientFirewallValues.failedChallengeCount) / constChallenge)
		currentBlacklistGiven := blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeFrontend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedChallengeTime",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedChallengeCount))
			return
		}
		timeSinceBlacklist = currentTime - clientFirewallValues.failedGoogleAuthTime
		blacklistExpGrowth = int64((clientFirewallValues.failedGoogleAuthCount) / constGoogleAuth)
		currentBlacklistGiven = blacklistExpGrowth * int64(time.Hour/time.Minute)
		if timeSinceBlacklist < currentBlacklistGiven {
			logInfo("(handshakeFrontend) blacklisted client attempting to connect:", r.RemoteAddr,
				"\n\tblacklistedFor:", string(timeSinceBlacklist),
				"\n\treason: failedGoogleAuth",
				"\n\tnumberOfOffenses:", string(clientFirewallValues.failedGoogleAuthCount))
			return
		}
		clientFirewallValues.RUnlock()
	} else {
		// intialize client with default values
		clientBlacklistMutex.Lock()
		clientBlacklist[client] = &FirewallClientValues{}
		clientBlacklistMutex.Unlock()
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

	go func() {
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
	}()

	defer func() {
		clientSessionCountMutex.Lock()
		_, ok := clientSessionCount[client]
		if ok {
			clientSessionCount[client]--
		} else {
			clientSessionCount[client] = 0
			logDebug("(handshakeFrontend) setting 'clientSessionCount' to ZERO",
				"for this client:", client)
		}
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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func kickoffFrontendInfoSync(host string) {

	////////////////////////////////////////////////////////////
	// start communication with frontend
	////////////////////////////////////////////////////////////

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("wss://"+host+":443/backend", nil)
	if checkError("(handshakeFrontend) unable to open websocket with frontend loadbalancer:", err) {
		return
	}
	defer conn.Close()

	// log client IP address
	logDebug("(handshakeFrontend) opening connection:", conn.RemoteAddr().String())
	defer logDebug("(handshakeFrontend) closing connection:", conn.RemoteAddr().String())

	////////////////////////////////////////////////////////////
	// initiate two-way challenge-response authentication
	////////////////////////////////////////////////////////////

	// define function to generate 256 bits of randomness
	newHMACKey := func() [32]byte {
		key := [32]byte{}
		_, err = io.ReadFull(cryptorand.Reader, key[:])
		if checkError("(handshakeFrontend) error while generating key", err) {
			conn.Close()
		}
		return key
	}

	// call above function and store value
	randomBytes := newHMACKey()
	logDebug("(handshakeFrontend) sending the following challenge:",
		"\n", base64.StdEncoding.EncodeToString(randomBytes[:]))

	// send random string to frontend
	err = conn.WriteMessage(websocket.BinaryMessage, randomBytes[:])
	if checkError("(handshakeFrontend) error sending message to frontend loadbalancer:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// parse incoming response
	////////////////////////////////////////////////////////////

	type incomingChallengeResponse struct {
		Challenge string `json:"challenge"`
		Response  string `json:"response"`
	}

	var inMsg incomingChallengeResponse
	err = conn.ReadJSON(&inMsg)
	if checkError("(handshakeFrontend) error receiving initial response from frontend loadbalancer:", err) {
		return
	}

	logDebug("(handshakeFrontend) received the following JSON challenge/response:\n",
		fmt.Sprintf("%s\n", inMsg))

	////////////////////////////////////////////////////////////
	// validate incoming response to our challenge
	////////////////////////////////////////////////////////////

	// define function to produce symmetric signature using a shared secret key
	generateHMAC := func(data []byte, key [32]byte) []byte {
		h := hmac.New(sha512.New512_256, key[:])
		h.Write(data)
		return h.Sum(nil)
	}

	// convert data to array
	var dataChallengeArray [32]byte
	copy(dataChallengeArray[:], randomBytes[:])

	// generate response hash based on challenge and our secret password
	var correctResponse []byte
	correctResponse = generateHMAC([]byte(authPassword), dataChallengeArray)

	// compare locally generated hash to received hash
	remoteHash, _ := base64.StdEncoding.DecodeString(inMsg.Response)
	if hmac.Equal(correctResponse, remoteHash) != true {
		logInfo("(handshakeFrontend) initial challenge-response failed:",
			"\n", base64.StdEncoding.EncodeToString(correctResponse),
			"\n", base64.StdEncoding.EncodeToString(remoteHash))
		return
	}

	////////////////////////////////////////////////////////////
	// respond to challenge sent to us
	////////////////////////////////////////////////////////////

	// decode into binary
	remoteChallenge, _ := base64.StdEncoding.DecodeString(inMsg.Challenge)

	// convert data to array
	var remoteChallengeArray [32]byte
	copy(remoteChallengeArray[:], remoteChallenge)

	// generate response hash based on challenge and our secret password
	ourResponse := generateHMAC([]byte(authPassword), remoteChallengeArray)
	logDebug("(handshakeFrontend) local hash for challenge sent by LoadBalancer:",
		"\n", base64.StdEncoding.EncodeToString(ourResponse))

	// send response to frontend
	err = conn.WriteMessage(websocket.BinaryMessage, ourResponse)
	if checkError("(handshakeFrontend) error sending message to frontend loadbalancer:", err) {
		return
	}

	////////////////////////////////////////////////////////////
	// initialize messages we'll be sending/receiving with frontend
	////////////////////////////////////////////////////////////

	var (
		inMsgTemp       = FrontendDataStruct{}
		inMsgReference  = FrontendDataStruct{}
		outMsgTemp      = FrontendDataStruct{}
		outMsgReference = FrontendDataStruct{}
	)

	////////////////////////////////////////////////////////////
	// define function that will be responsible for sending/receiving
	////////////////////////////////////////////////////////////

	periodicUpdate := func() bool {

		////////////////////////////////////////////////////////////
		// zeroing out temp messages (important)
		////////////////////////////////////////////////////////////

		inMsgTemp = FrontendDataStruct{}
		outMsgTemp = FrontendDataStruct{}

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

		for i := 0; i < len(frontendData.ListOfGalaxies); i++ {
			outMsgTemp.ListOfGalaxies[i].GalaxyName = frontendData.ListOfGalaxies[i].GalaxyName
			outMsgTemp.ListOfGalaxies[i].GlobalX = frontendData.ListOfGalaxies[i].GlobalX
			outMsgTemp.ListOfGalaxies[i].GlobalY = frontendData.ListOfGalaxies[i].GlobalY
		}

		outMsgTemp.ListOfGalaxies = frontendData.ListOfGalaxies
		outMsgTemp.OtherFrontends = frontendData.OtherFrontends
		outMsgTemp.OtherFrontends = appendStringIfMissing(outMsgTemp.OtherFrontends, myIPaddr)

		frontendDataMutex.RUnlock()

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
			outMsgTemp = FrontendDataStruct{Message: sameaslasttime}
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

		frontendDataMutex.RLock()
		copyOfFrontendSlice := make([]string, len(frontendData.OtherFrontends))
		copy(copyOfFrontendSlice, frontendData.OtherFrontends)
		copyOfGalaxySlice := make([]galInfo, len(frontendData.ListOfGalaxies))
		copy(copyOfGalaxySlice, frontendData.ListOfGalaxies)
		frontendDataMutex.RUnlock()

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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
