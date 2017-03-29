package main

import (
	"bytes"
	"crypto/tls"
	"regexp"
	"sync"

	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"google.golang.org/api/oauth2/v2"
	//"golang.org/x/crypto/acme/autocert"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// defining global variables
var (
	inputFilename    string
	outputFilename   string
	googleAudience   string
	authPassword     string
	loadBalanceBool  bool
	createGalaxyBool bool
	debugModeBool    bool
)

func init() {
	// define commandline flags to parse for
	inputFilePtr := flag.String("input", "", "specify data source (either json or blob)")
	outputFilePtr := flag.String("output", "", "specify export destination (either json or blob)")
	defaultAudience := "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	googleAudiencePtr := flag.String("audience", defaultAudience, "specify google oauth 'audience'")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes (not optional)")
	loadBalanceBoolPtr := flag.Bool("loadbalance", false, "enable frontend loadbalancer mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable generation of new Galaxy (dangerous)")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")

	// pull flags that were attached to commandline
	flag.Parse()

	// convert pointers to values and store in global variables
	inputFilename = *inputFilePtr
	outputFilename = *outputFilePtr
	googleAudience = *googleAudiencePtr
	authPassword = *authPasswordPtr
	loadBalanceBool = *loadBalanceBoolPtr
	createGalaxyBool = *createGalaxyBoolPtr
	debugModeBool = *debugModeBoolPtr
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// defining global variables
var (
	randomSeed     *rand.Rand
	currentGalaxy  Galaxy
	futureGalaxy   Galaxy
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
)

func main() {
	// initializing main data structures, locks, etc
	currentGalaxy = Galaxy{}
	//futureGalaxy := Galaxy{}

	// branch here based on command line invocation
	if loadBalanceBool == true {
		startLoadBalancingMode()
		log.Fatalln("PROGRAM: LoadBalancingMode is terminating... (and wasn't supposed to)")
		waitForLogging.Wait()
		os.Exit(1)
	} else {
		startBackendMode()
		log.Fatalln("PROGRAM: BackendMode is finished executing")
		waitForLogging.Wait()
		os.Exit(1)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startLoadBalancingMode() {

	// setup httpMux to avoid leaking default handlers
	muxSecure := http.NewServeMux()

	// to enable perfect forward secrecy it's necessary to add 'Strict-Transport-Security'
	// since we are not writing out own FileServer, we'll need to add Headers this way
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

	// setup the loadbalance handler ... please note that the security headers are added in
	// the 'loadBalancingWebSocket' function itself unlike earlier with the fileserver handler
	muxSecure.HandleFunc("/loadbalance", loadBalancingWebSocket)

	// defining various security parameters used to specify accepted ciphersuites
	cfg := &tls.Config{
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
	}

	// tying the above security parameters to the HTTP handlers associated with 'muxSecure'
	srvSecure := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Addr:         ":443",
		Handler:      muxSecure,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// define waitgroup to be used to stop premature termination
	var lbWaitGroup sync.WaitGroup

	// Serve HTTPS requests with config defined above
	lbWaitGroup.Add(1)
	go func() {
		errServeTLS := srvSecure.ListenAndServeTLS("server.rsa.crt", "server.rsa.key")
		if checkError("(ListenAndServeTLS) Something went wrong:", errServeTLS) {
			waitForLogging.Wait()
			os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// define non secure HTTP parameters
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

	// redirect all "/" HTTP requests to HTTPS "/" homepage
	lbWaitGroup.Add(1)
	go func() {
		errServeInsecure := srvInsecure.ListenAndServe()
		if checkError("(ListenAndServeInsecure) Something went wrong:", errServeInsecure) {
			waitForLogging.Wait()
			os.Exit(1)
		}
		lbWaitGroup.Done()
	}()

	// use waitgroup to stop permature termination unless all the above goroutines are finished executing
	lbWaitGroup.Wait()
}

func loadBalancingWebSocket(w http.ResponseWriter, r *http.Request) {

	// to enable perfect forward secrecy it's necessary to add 'Strict-Transport-Security' header
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

	upgrader := websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(lbsocket) upgrade error:", err) == true {
		return
	}
	defer conn.Close()

	logDebug("(lbsocket) opening socket:", string(r.RemoteAddr))

	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}

	type PotentialServers struct {
		GalaxyName string
		ServerList []string
	}
	type outgoingMessage struct {
		Servers []PotentialServers
	}

	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if checkError("(lbsocket) error receiving message:", err) == true {
		conn.Close()
	}

	logDebug("(lbsocket) received the following message:", fmt.Sprintf("%s", inMsg))

	var httpClient = &http.Client{}
	oauth2Service, err := oauth2.New(httpClient)
	checkError("(lbsocket) error contacting google:", err)

	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.IDToken)
	tokenInfo, err := tokenInfoCall.Do()
	checkError("(lbsocket) error parsing incoming google response:", err)

	rawJSON, err := tokenInfo.MarshalJSON()
	checkError("(lbsocket) error MarshalJSON'ing google response:", err)

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, rawJSON, "", "    ")
	checkError("(lbsocket) error jsonIndent'ing google response:", err)

	logDebug("(lbsocket) received the following message from google:", string(prettyJSON.Bytes()))

	//err = conn.WriteMessage(websocket.BinaryMessage, lbgalaxy)
	//if checkError("(lbsocket) error sending message:", err) == true {
	//	conn.Close()
	//}

	//logDebug("(lbsocket) just sent", fmt.Sprintf("%v", binary.Size(lbgalaxy)), "bytes")

	conn.Close()
	logDebug("(lbsocket) closing socket:", string(r.RemoteAddr))
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startBackendMode() {

	// handle input file parameters
	// is the inputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	var doneMatching bool
	if inputFilename == "" {
		doneMatching = true
		logDebug("(backend_init) no inputfilename specified - skipping data import process")
	} else {
		doneMatching = false
	}
	// skip forward if doneMatching is 'true'
	// otherwise parse filename to determine if it is a 'blob' file
	// and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*blob$", inputFilename)
		if foundMatch == true {
			doneMatching = true
			logInfo("(backend_init) parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFilename)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln("data.blob not found")
				} else {
					log.Fatalln("Error reading file:", inputFilename, err)
				}
			}
			if err := proto.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// skip forward if doneMatching is 'true'
	// otherwise parse filename to determine if it is a 'json' file
	// and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*json$", inputFilename)
		if foundMatch == true {
			doneMatching = true
			logInfo("(backend_init) parsing blob inputfilename")
			in, err := ioutil.ReadFile(inputFilename)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln(inputFilename, "not found")
				} else {
					log.Fatalln("Error reading file:", inputFilename, err)
				}
			}
			if err := json.Unmarshal(in, &currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// if none of the above logic was successful - we have a fatal error on our hands
	if doneMatching == false {
		log.Fatalln("(backend_init) unknown inputfilename - skipping data import process")
	}

	// create websocket with loadbalancer frontend
	var dialer *websocket.Dialer
	conn, _, err := dialer.Dial("ws://hyperion5.com:81", nil)
	if checkError("(backend_init) unable to open websocket with frontend loadbalancer:", err) {
		waitForLogging.Wait()
		os.Exit(1)
	}

	// generate 128 random bytes
	randomBytes := make([]byte, 128)
	_, err = rand.Read(randomBytes)
	checkError("(backend_init) error while generating bytes:", err)

	// convert randomBytes to randomString and print
	if debugModeBool == true {
		randomString := base64.URLEncoding.EncodeToString(randomBytes)
		logDebug("(backend_init) sent the following challenge: ", randomString)
	}

	// send random string to frontend
	err = conn.WriteMessage(websocket.TextMessage, []byte(randomBytes))
	if checkError("(backend_init) error sending message to frontend loadbalancer:", err) {
		waitForLogging.Wait()
		os.Exit(1)
	}

	// wait for response from websocket frontend
	_, _, err = conn.ReadMessage()
	if checkError("(backend_init) error receiving message from frontend loadbalancer:", err) {
		waitForLogging.Wait()
		os.Exit(1)
	}

	// main game loop
	// lots of concurrency here (locking, waitgroups, etc)
	//for {
	//
	//}

	//i := 0
	//for {
	//	i++
	//	for _, ss := range currentGalaxy.Solarsystemslice {
	//		for _, pl := range ss.Planetslice {
	//			pl.Population = growPopulation(pl.Population, pl.Food)
	//		}
	//	}
	//	if i == 100 {
	//		break
	//	}
	//}

}

func backendWebSocket(w http.ResponseWriter, r *http.Request) {
	//upgrader := websocket.Upgrader{
	//	ReadBufferSize:  1024,
	//	WriteBufferSize: 1024,
	//}
}

func gracefullyCloseBackend() {

	// handle output file parameters
	// is the outputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	foundMatch, _ := regexp.MatchString(".*blob$", inputFilename)
	if inputFilename == "" || foundMatch {
		protoOut, err := proto.Marshal(&currentGalaxy)
		if err != nil {
			log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
		}
		if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	} else {
		jsonOut, err := json.MarshalIndent(currentGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(outputFilename, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func absInt64(val int64) int64 {
	if val < 0 {
		val = -val
	}
	return val
}

func commaFormatUint(n uint64) string {
	in := strconv.FormatUint(n, 10)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}
	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func mulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

const mostNegative = -(mostPositive + 1)
const mostPositive = 1<<63 - 1

func signedMulOverflows(a, b int64) bool {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return false
	}
	if a == mostNegative || b == mostNegative {
		return true
	}
	c := a * b
	return c/b != a
}

func checkError(message string, err error) bool {
	waitForLogging.Add(1)
	var errorFound bool
	if err != nil {
		errorFound = true
		go func() {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			err := err.Error() // convert err to string
			loggingMutex.Lock()
			color.Red(t + " " + message + err)
			loggingMutex.Unlock()
			waitForLogging.Done()
		}()
	} else {
		errorFound = false
	}
	return errorFound
}

func logDebug(arrayOfStrings ...string) {
	waitForLogging.Add(1)
	go func() {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		if debugModeBool == true {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			loggingMutex.Lock()
			color.Cyan(t + " " + message)
			loggingMutex.Unlock()
			waitForLogging.Done()
		}
	}()
}

func logInfo(arrayOfStrings ...string) {
	waitForLogging.Add(1)
	go func() {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		loggingMutex.Lock()
		color.White(t + " " + message)
		loggingMutex.Unlock()
		waitForLogging.Done()
	}()
}

func growPopulation(currentPopulation float64, foodSupply int32) float64 {
	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	referencePopulation := float64(10000) // population if foodSupply == 0
	maxPercentChange := .005              // maximum % change allowed in single turn (.0005 is standard)

	N := currentPopulation
	logDebug(commaFormatUint(uint64(N)))

	randomness := referencePopulation * (1 + rnum.NormFloat64()*.1)
	K := randomness * math.Pow(foodSupplyDbl, 3.4)

	logisticPercentChange := (K - N) / K

	switch {
	case logisticPercentChange < -1:
		logisticPercentChange = -1
	case logisticPercentChange > 1:
		logisticPercentChange = 1
	case logisticPercentChange == 0:
		logisticPercentChange = .000001
	}

	currentPopulation = currentPopulation + (N*maxPercentChange)*logisticPercentChange
	return currentPopulation
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// planettype
const (
	TERRAN int32 = iota
	DESERT
	VOLCANIC
	ICE
	BARREN
	FERROUS
	GREENHOUSE
	OCEANIC
	ASTEROID
	DWARF
)

// spaceshiptype
const (
	SCOUT int32 = iota
	COLONY
	CARRIER
	BATTLESHIP
	TITAN
)

// solarsystemtype
const (
	BLUEGIANT int32 = iota
	YELLOWGIANT
	REDGIANT
	WHITEDWARF
	NEUTRONSTAR
	BLACKHOLE
)

// spaceshipstatus
const (
	OPERATIONAL int32 = iota
	LIGHTLYDAMAGED
	HEAVILYDAMAGED
	SEVERELYDAMAGED
	HANGINGBYATHREAD
)

// galaxytype
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
