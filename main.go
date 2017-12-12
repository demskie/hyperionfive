package main

import (
	"bytes"
	"container/ring"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"runtime"
	"sync"

	"flag"
	"math/rand"
	"os"
	"strconv"
	"time"

	_ "net/http/pprof"

	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

func main() {
	const fName = "(main)"
	switch {
	case GLOBAL.frontendBool == true:
		startFrontendMode() // frontend.go
	case GLOBAL.backendBool == true:
		startBackendMode() // backend.go
	default:
		logError(fName, "no operating mode was specified")
	}
	suddenExit()
}

func init() {
	const fName = "(init)"

	// sorry not sorry
	GLOBAL = globalVars{}

	GLOBAL.logInfoChan = make(chan loggingMessage, oneMillion)

	GLOBAL.serverPriority = generateRandomUint64()
	GLOBAL.numberOfCores = 0
	GLOBAL.loggingMutex = sync.Mutex{}
	GLOBAL.waitForCores = sync.WaitGroup{}
	GLOBAL.exitProgram = make(chan struct{})

	logInfo(fName, "logging initialized")
	logInfo(fName, fmt.Sprintf("serverPriority=%d", GLOBAL.serverPriority))
	go handleLogging()

	GLOBAL.clientBlacklist = clientBlacklist{}
	GLOBAL.clientBlacklist.LiveConnections = map[string][]string{} // remoteIP to functionNames
	//GLOBAL.clientBlacklist.TotalSessionCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedChallengeTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedGoogleAuthCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedGoogleAuthTime = map[string]time.Time{}
	GLOBAL.clientBlacklist.FailedSessionLimitCount = map[string]uint16{}
	GLOBAL.clientBlacklist.FailedSessionLimitTime = map[string]time.Time{}

	GLOBAL.turningDataSignal = make(chan struct{})
	GLOBAL.newTickSignal = make(chan struct{})
	GLOBAL.mutatingDataBlocker = make(chan struct{})
	GLOBAL.allServersReadySignal = make(chan struct{})
	GLOBAL.currentGalaxy = Galaxy{}
	GLOBAL.futureGalaxy = Galaxy{}
	GLOBAL.uniqueRequestID = make(chan int64, 9999)
	GLOBAL.bufferedRequests = make([]incomingClientRequest, 0, oneMillion)
	GLOBAL.bufferedRequestsMutex = sync.RWMutex{}

	GLOBAL.backendData = backendData{}
	GLOBAL.backendData.Frontends = []string{}
	GLOBAL.backendData.Backends = []string{}
	GLOBAL.backendData.ClientIDs = map[string][]string{}
	GLOBAL.backendData.SystemBytes = map[string][]byte{}
	GLOBAL.backendData.SystemHashes = [16]map[string]map[string][4]byte{}
	GLOBAL.backendData.ParsedPlayerPtrs = map[string][]*SolarSystem{}
	GLOBAL.backendData.Priorities = map[string]uint64{}
	GLOBAL.backendData.Latencies = map[string]*ring.Ring{}
	GLOBAL.backendData.TickTime = 0
	GLOBAL.backendData.NeighborTickTimes = map[string]uint64{}
	GLOBAL.backendData.ReceivedMessageFrom = []string{}
	GLOBAL.backendData.ConfirmedRequestIDs = map[string][]int64{}

	GLOBAL.frontendData = frontendData{}
	GLOBAL.frontendData.GalGlobalX = map[string]int64{}
	GLOBAL.frontendData.GalGlobalY = map[string]int64{}
	GLOBAL.frontendData.GalClientIDs = map[string][]string{}
	GLOBAL.frontendData.GalBackends = map[string][]string{}
	GLOBAL.frontendData.BackendLoad = map[string]int{}
	GLOBAL.frontendData.OtherFrontends = []string{}
	GLOBAL.frontendData.Message = ""
	GLOBAL.frontendData.AnotherMessage = ""

	// define commandline flags to parse
	galaxyNamePtr := flag.String("galaxyname", "", "specify galaxy name")
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "override export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "override google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	staticFrontendPtr := flag.String("staticfrontend", "", "specify static frontend address")
	localInterfacePtr := flag.String("interface", "", "override local interface to use")
	localAddressPtr := flag.String("localip", "", "override local ip to use")
	frontendBoolPtr := flag.Bool("frontend", false, "enable frontend mode")
	backendBoolPtr := flag.Bool("backend", false, "enable backend mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")
	singleThreadedBoolPtr := flag.Bool("singlethreaded", false, "force singlethreaded behavior")
	testModeBoolPtr := flag.Bool("test", false, "enable testing mode")

	// parse the arguments
	flag.Parse()
	GLOBAL.galaxyName = *galaxyNamePtr
	GLOBAL.inputFile = *inputFilePtr
	GLOBAL.outputFile = *outputFilePtr
	GLOBAL.googleAudience = *googleAudiencePtr
	GLOBAL.authPassword = *authPasswordPtr
	GLOBAL.staticFrontend = *staticFrontendPtr
	GLOBAL.localInterface = *localInterfacePtr
	GLOBAL.localAddress = *localAddressPtr
	GLOBAL.frontendBool = *frontendBoolPtr
	GLOBAL.backendBool = *backendBoolPtr
	GLOBAL.createGalaxyBool = *createGalaxyBoolPtr
	GLOBAL.debugModeBool = *debugModeBoolPtr
	GLOBAL.singleThreadedBool = *singleThreadedBoolPtr
	GLOBAL.testModeBool = *testModeBoolPtr

	if GLOBAL.googleAudience == "" {
		GLOBAL.googleAudience = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}

	// sanity check that a password was provided
	if GLOBAL.authPassword == "" {
		logInfo(fName, "no password provided, using default password")
		GLOBAL.authPassword += "uVXMHpASbDngPe41pUP1"
		GLOBAL.authPassword += "CKwRCU6ZGap6IxQ6OQaM"
		GLOBAL.authPassword += "AtJgeaIhHLzO0IibxE8B"
		GLOBAL.authPassword += "7aYOpAXkC388SRg4G3jO"
		GLOBAL.authPassword += "kkgrBnOOUPxnT6QopsFH"
		GLOBAL.authPassword += "a0F7qFScNxYuCaiRvii3"
		GLOBAL.authPassword += "WvhhNNsFaRupwZc7SXXw"
		GLOBAL.authPassword += "cPk93TWZs0kllqVhCN0G"
		GLOBAL.authPassword += "9oKiFpa0p7Mfwd8CqHiz"
		GLOBAL.authPassword += "3ghjZw6FI17DjdObCXSF"
	}

	// singlethreaded mode for debugging (and emergencies)
	if GLOBAL.singleThreadedBool {
		GLOBAL.numberOfCores = 1
	} else {
		GLOBAL.numberOfCores = runtime.NumCPU()
	}
	logInfo(fName, fmt.Sprintf("using %v threads", GLOBAL.numberOfCores))
}

func suddenExit() {
	i := 1
	for len(GLOBAL.logInfoChan) != 0 {
		time.Sleep(time.Second)
		if i > 30 {
			break
		}
		i++
	}
	GLOBAL.loggingMutex.Lock()
	t := time.Now().UTC().Format("2006-01-02 15:04:05")
	color.Red(t + " (main) program has finished executing")
	GLOBAL.loggingMutex.Unlock()
	os.Exit(1)
}

func handleInputParameters(fName string) {
	// is the inputfilename empty?
	// if so set doneMatching to 'true' so that we can skip later logic
	var doneMatching bool
	if GLOBAL.inputFile == "" {
		doneMatching = true
		logDebug(fName, "no inputfilename specified - skipping data import process")
	} else {
		doneMatching = false
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'blob' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*blob$", GLOBAL.inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName, "parsing blob inputfilename")
			in, err := ioutil.ReadFile(GLOBAL.inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln("data.blob not found")
				} else {
					log.Fatalln("Error reading file:", GLOBAL.inputFile, err)
				}
			}
			if err := proto.Unmarshal(in, &GLOBAL.currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// skip forward if doneMatching is 'true'
	//   otherwise parse filename to determine if it is a 'json' file
	//   and import that file if that is the case
	if doneMatching == false {
		foundMatch, _ := regexp.MatchString(".*json$", GLOBAL.inputFile)
		if foundMatch == true {
			doneMatching = true
			logInfo(fName, "parsing blob inputfilename")
			in, err := ioutil.ReadFile(GLOBAL.inputFile)
			if err != nil {
				if os.IsNotExist(err) {
					log.Fatalln(GLOBAL.inputFile, "not found")
				} else {
					log.Fatalln("Error reading file:", GLOBAL.inputFile, err)
				}
			}
			if err := json.Unmarshal(in, &GLOBAL.currentGalaxy); err != nil {
				log.Fatalln("Failed to parse galaxy file:", err)
			}
		}
	}
	// if none of the above logic was successful - we have a fatal error on our hands
	if doneMatching == false {
		log.Fatalln(fName, "unknown inputfilename. skipping data import process")
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func getWebsocketDialer() (dialer *websocket.Dialer) {
	dialer = &websocket.Dialer{}
	defer func() {
		logDebug(fmt.Sprintf("*websocket.Dialer = %+v", dialer))
	}()
	if GLOBAL.localInterface != "" {
		iface, err := net.InterfaceByName(GLOBAL.localInterface)
		if checkError(err) == false {
			addrs, anotherError := iface.Addrs()
			if checkError(anotherError) == false {
				tcpAddr := &net.TCPAddr{
					IP: addrs[0].(*net.IPNet).IP,
				}
				logDebug("using", iface.Name, "and",
					addrs[0].(*net.IPNet).IP.String())
				netDialer := &net.Dialer{
					LocalAddr: tcpAddr,
					Deadline:  time.Now().Add(time.Second * 30),
				}
				logDebug(fmt.Sprintf("netDialer = %+v", netDialer))
				dialer = &websocket.Dialer{
					NetDial: netDialer.Dial,
				}
			}
		}
	}
	if GLOBAL.localAddress != "" {
		ifaces, err := net.Interfaces()
		if checkError(err) == false {
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if checkError(err) {
					continue
				}
				for _, addr := range addrs {
					if checkError(err) {
						continue
					}
					ipAddr := addr.(*net.IPNet).IP
					if ipAddr.String() == GLOBAL.localAddress {
						tcpAddr := &net.TCPAddr{
							IP: addr.(*net.IPNet).IP,
						}
						logDebug("using", iface.Name, "and",
							ipAddr.String())
						netDialer := &net.Dialer{
							LocalAddr: tcpAddr,
							Deadline:  time.Now().Add(time.Second * 30),
						}
						logDebug(fmt.Sprintf("netDialer = %+v", netDialer))
						dialer = &websocket.Dialer{
							NetDial: netDialer.Dial,
						}
						return dialer
					}
				}
			}
		}
	}
	return dialer
}

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

func appendStringIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func removeStringIfPresent(slice []string, s string) []string {
	for i, ele := range slice {
		if ele == s {
			slice[i] = slice[len(slice)-1]
			slice = slice[:len(slice)-1]
		}
	}
	return slice
}

func generateRandomUint64() uint64 {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	rnum := rand.New(seed())
	return rnum.Uint64()
}

func getCryptoRnum() *rand.Rand {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	return rand.New(seed())
}

func getPosOrNegOne() int {
	var randomNum int8
	b := make([]byte, 1)
	cryptorand.Read(b)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &randomNum)
	if randomNum >= 0 {
		return 1
	}
	return -1
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
