package main

import (
	"bytes"
	"sync"

	"encoding/binary"
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

//type base uint8

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

// global variables
var (
	theGalaxy    Galaxy
	theNewGalaxy Galaxy
	lbgalaxy     []byte
	debugMode    bool
	loggingMutex sync.Mutex
)

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
	var errorFound bool
	go func() {
		if err != nil {
			errorFound = true
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			err := err.Error() // convert err to string
			loggingMutex.Lock()
			color.Red(t + " " + message + err)
			loggingMutex.Unlock()
		} else {
			errorFound = false
		}
	}()
	return errorFound
}

func logDebug(arrayOfStrings ...string) {
	go func() {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		if debugMode == true {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			loggingMutex.Lock()
			color.White(t + " " + message)
			loggingMutex.Unlock()
		}
	}()
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

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
	//println("K                 = " + commaFormatUint(uint64(K)))

	logisticPercentChange := (K - N) / K

	switch {
	case logisticPercentChange < -1:
		//println("   negative precision loss", logisticPercentChange)
		logisticPercentChange = -1
	case logisticPercentChange > 1:
		//println("   positive precision loss", logisticPercentChange)
		logisticPercentChange = 1
	case logisticPercentChange == 0:
		//println("   zero value             ", logisticPercentChange)
		logisticPercentChange = .000001
	}

	currentPopulation = currentPopulation + (N*maxPercentChange)*logisticPercentChange
	return currentPopulation
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func startLoadBalancingMode() {

	inputFilename := "data.blob"
	const dataBlob = "data.blob"

	var err error
	if inputFilename == dataBlob {
		lbgalaxy, err = ioutil.ReadFile(dataBlob)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("data.blob not found\n")
				os.Exit(1)
			} else {
				log.Fatalln("Error reading file:", err)
			}
		}
		if err := proto.Unmarshal(lbgalaxy, &theGalaxy); err != nil {
			log.Fatalln("Failed to parse galaxy file:", err)
		}
	}

	flsrv := http.FileServer(http.Dir("D:\\OneDrive\\GoPrototypes\\src\\GOSERVER\\websiteroot\\"))
	http.Handle("/", flsrv)

	http.HandleFunc("/loadbalance", lbsocket)
	//http.HandleFunc("/oauth2callback", handleOAuth2Callback)
	//http.Handle("/loadbalance", websocket.Handler(lbsocket))

	//err := http.ListenAndServe(":80", nil)
	//if err != nil {
	//	log.Fatalln("ListenAndServe: ", err)
	//}
	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println(srv.ListenAndServe())
	log.Fatalln("LoadBalancing mode is terminating... (and wasn't supposed to)")
	os.Exit(99)
}

func lbsocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if checkError("(lbsocket) upgrade error:", err) == true {
		return
	}
	defer conn.Close()

	//ip, srcport_in, err := net.SplitHostPort(r.RemoteAddr)
	//if err != nil {
	//	color.Red("(lbsocket) SplitHostPort error:", err)
	//}
	//color.Magenta("%s - (lbsocket) - opening socket: %s", time.Now().Format(time.RFC850), r.RemoteAddr)

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

	//log.Println(verifyIDToken(inMsg.IDToken))

	//result := map[string]string{"Andromeda-187": "192.168.1.26", "Milkyway-832": "192.168.1.99"}

	//var outMsg outgoingMessage
	//outMsg.Servers = append(outMsg.Servers, PotentialServers{GalaxyName: "Milkyway-832", ServerList: []string{"127.0.0.1", "hahahaha"}})
	//outMsg.Servers = append(outMsg.Servers, PotentialServers{GalaxyName: "Andromeda-187", ServerList: []string{"10.0.0.99", "192.168.1.1"}})

	//fmt.Println(theGalaxy.Name)
	//for _, ss := range theGalaxy.Solarsystemslice {
	//	fmt.Println("  ", ss.Name)
	//	for _, pl := range ss.Planetslice {
	//		fmt.Println("    ", pl.Name)
	//		fmt.Println("      ", pl.Population)
	//	}
	//}

	err = conn.WriteMessage(websocket.BinaryMessage, lbgalaxy)
	if checkError("(lbsocket) error sending message:", err) == true {
		conn.Close()
	}

	logDebug("(lbsocket) just sent", fmt.Sprintf("%v", binary.Size(lbgalaxy)), "bytes")

	conn.Close()
	logDebug("(lbsocket) closing socket:", string(r.RemoteAddr))

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	debugMode = true
}

func main() {

	// define OS arguments for parsing
	inputFilePtr := flag.String("input", "data.blob", "use specified file as data source (optional)")
	outputFilePtr := flag.String("output", "", "specify json file for export (optional)")
	loadBalanceBoolPtr := flag.Bool("loadbalance", false, "act as frontend loadbalancer (optional)")
	//createGalaxyBoolPtr := flag.Bool("generate", false, "generate a new Galaxy (optional and dangerous)")
	flag.Parse()
	inputFilename := *inputFilePtr
	outputFilename := *outputFilePtr
	loadBalanceBool := *loadBalanceBoolPtr
	//createGalaxyBool := *createGalaxyBool

	theGalaxy := Galaxy{}
	//theGalaxyRWM := &sync.RWMutex{}
	//theNewGalaxy := Galaxy{}
	//theNewGalaxyRWM := &sync.RWMutex{}

	//http.HandleFunc("/oauth2callback", handleOAuth2Callback)

	if loadBalanceBool == true {
		startLoadBalancingMode()
	}

	const dataBlobDefault = "data.blob"

	// run various sanity checks and/or print error messages here
	if outputFilename == dataBlobDefault {
		log.Fatalln("Improper use of the 'output' parameter. 'Output' is used to specify where JSON should go. This parameter is optional.")
	}

	if inputFilename == dataBlobDefault {
		in, err := ioutil.ReadFile(dataBlobDefault)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalln("data.blob not found")
			} else {
				log.Fatalln("Error reading file:", dataBlobDefault, err)
			}
		}
		if err := proto.Unmarshal(in, &theGalaxy); err != nil {
			log.Fatalln("Failed to parse galaxy file:", err)
		}
	} else {
		in, err := ioutil.ReadFile(inputFilename)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalln(inputFilename, "not found")
			} else {
				log.Fatalln("Error reading file:", inputFilename, err)
			}
		}
		if err := json.Unmarshal(in, &theGalaxy); err != nil {
			log.Fatalln("Failed to parse galaxy file:", err)
		}
	}

	i := 0
	for {
		i++
		for _, ss := range theGalaxy.Solarsystemslice {
			for _, pl := range ss.Planetslice {
				pl.Population = growPopulation(pl.Population, pl.Food)
			}
		}
		//fmt.Println(theGalaxy.Name)
		//for _, ss := range theGalaxy.Solarsystemslice {
		//	fmt.Println("  ", ss.Name)
		//	for _, pl := range ss.Planetslice {
		//		fmt.Println("    ", pl.Name)
		//		fmt.Println("      ", pl.Population)
		//	}
		//}
		if i == 100 {
			break
		}
	}

	// Create default values for Galaxy and attempt to Parse file
	//theGalaxy = Galaxy{GlobalX: 526, GlobalY: -283, Name: "Milkyway-832"}
	//someSystems := make([]*SolarSystem, 0)
	//someSystems = append(someSystems,
	//	&SolarSystem{InterX: 3293, InterY: 99, Name: "Sol-137",},
	//	&SolarSystem{InterX: -233, InterY: 10039, Name: "Xioami-137"})
	//theGalaxy.Solarsystemslice = someSystems

	// Write new galaxy to data.blob
	protoOut, err := proto.Marshal(&theGalaxy)
	if err != nil {
		log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
	}
	if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
		log.Fatalln("Failed to write Galaxy:", err)
	}

	// Write new galaxy to local json file
	if outputFilename != "" {
		jsonOut, err := json.MarshalIndent(theGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(outputFilename, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}

	//fmt.Println(theGalaxy)

	//http.Handle("/", http.FileServer(http.Dir("D:\\OneDrive\\JSCLIENT")))
	//http.HandleFunc("/ws", handleWebsocket)
	//err = http.ListenAndServe(":80", nil)
	//if err != nil {
	//	log.Fatal("ListenAndServe: ", err)
	//}
}
