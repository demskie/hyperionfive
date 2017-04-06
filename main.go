package main

import (
	"sync"

	"flag"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// defining various global variables
var (
	inputFilename    string
	outputFilename   string
	googleAudience   string
	authPassword     string
	staticFrontend   string
	loadBalanceBool  bool
	createGalaxyBool bool
	debugModeBool    bool

	clientBlacklist         map[string]FirewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex

	randomSeed     *rand.Rand
	currentGalaxy  Galaxy
	futureGalaxy   Galaxy
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
)

// defining global constants for internal firewall
const (
	constChallenge     = 1
	constGoogleAuth    = 3
	constClientSession = 25
)

// FirewallClientValues defines values for firewalling
type FirewallClientValues struct {
	mutex                   sync.RWMutex
	failedChallengeCount    uint16
	failedChallengeTime     int64
	failedGoogleAuthCount   uint16
	failedGoogleAuthTime    int64
	failedSessionLimitCount uint16
	failedSessionLimitTime  int64
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func init() {

	// initializing global IP blacklist
	clientBlacklist = map[string]FirewallClientValues{}

	// initializing global IP session count
	clientSessionCount = make(map[string]uint16)

	// define commandline flags to parse for
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "specify export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "specify google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	staticFrontendPtr := flag.String("staticfrontend", "", "specify static frontend address")
	loadBalanceBoolPtr := flag.Bool("loadbalance", false, "enable frontend loadbalancer mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")

	// pull flags attached to CLI invocation
	flag.Parse()

	// handle default and/or invalid parameters
	if *googleAudiencePtr == "" {
		*googleAudiencePtr = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}
	if *authPasswordPtr == "" {
		logInfo("(init) FATALERROR: no password provided")
		waitForLogging.Wait()
		os.Exit(1)
	}

	// convert pointers to values and store in global variables
	inputFilename = *inputFilePtr
	outputFilename = *outputFilePtr
	googleAudience = *googleAudiencePtr
	authPassword = *authPasswordPtr
	staticFrontend = *staticFrontendPtr
	loadBalanceBool = *loadBalanceBoolPtr
	createGalaxyBool = *createGalaxyBoolPtr
	debugModeBool = *debugModeBoolPtr

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func main() {

	// branch here based on command line invocation
	if loadBalanceBool == true {
		startFrontendMode()
		log.Fatalln("PROGRAM: FrontendMode is finished executing")
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
