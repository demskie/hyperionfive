package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"runtime"
	"sync"

	"flag"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/fatih/color"
	"github.com/golang/protobuf/proto"
)

func init() {

	// start logging routine
	logInfoChan = make(chan string)
	go handleLogging()
	logInfo("(initLogInfo) - init completed")

	// initializing global IP blacklist
	clientBlacklist = map[string]*FirewallClientValues{}

	// initializing global IP session count
	clientSessionCount = make(map[string]uint16)

	// initializing galaxyInfo
	galaxyInfo = map[string]*GalaxyInfoStruct{}

	// define commandline flags to parse for
	galaxyNamePtr := flag.String("galaxyname", "", "specify galaxy name")
	inputFilePtr := flag.String("input", "", "specify data source, either json or blob")
	outputFilePtr := flag.String("output", "", "override export destination, either json or blob")
	googleAudiencePtr := flag.String("audience", "", "override google oauth audience")
	authPasswordPtr := flag.String("password", "", "specify password for authentication purposes")
	staticFrontendPtr := flag.String("staticfrontend", "", "specify static frontend address")
	frontendBoolPtr := flag.Bool("frontend", false, "enable frontend mode")
	backendBoolPtr := flag.Bool("backend", false, "enable backend mode")
	createGalaxyBoolPtr := flag.Bool("generate", false, "enable creation of new Galaxy, dangerous!")
	debugModeBoolPtr := flag.Bool("debug", false, "enable debug logging")
	singleThreadedBoolPtr := flag.Bool("singlethreaded", false, "force singlethreaded behavior")

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
	switch {
	case *backendBoolPtr == false && *frontendBoolPtr == false:
		logInfo("(init) FATALERROR: no backend/frontend mode specified")
		waitForLogging.Wait()
		os.Exit(1)
	case *backendBoolPtr == true && *frontendBoolPtr == true:
		logInfo("(init) FATALERROR: BOTH backend/frontend mode specified - DoesNotCompute")
		waitForLogging.Wait()
		os.Exit(1)
	}

	// singlethreaded mode for debugging (and emergencies)
	if singleThreadedBool {
		numberOfCores = 1
	} else {
		numberOfCores = runtime.NumCPU()
	}

	// convert pointers to values and store in global variables
	galaxyName = *galaxyNamePtr
	inputFilename = *inputFilePtr
	outputFilename = *outputFilePtr
	googleAudience = *googleAudiencePtr
	authPassword = *authPasswordPtr
	staticFrontend = *staticFrontendPtr
	frontendBool = *frontendBoolPtr
	backendBool = *backendBoolPtr
	createGalaxyBool = *createGalaxyBoolPtr
	debugModeBool = *debugModeBoolPtr
	singleThreadedBool = *singleThreadedBoolPtr

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	if frontendBool == true {
		startFrontendMode()
		for len(logInfoChan) != 0 {
			time.Sleep(time.Second)
		}
		loggingMutex.Lock()
		fmt.Println("")
		fmt.Println("##############################################")
		fmt.Println("#     Frontend has finished executing!!!     #")
		fmt.Println("##############################################")
		fmt.Println("")
		loggingMutex.Unlock()
		time.Sleep(1 * time.Hour)
		os.Exit(1)
	} else {
		startBackendMode()
		for len(logInfoChan) != 0 {
			time.Sleep(time.Second)
		}
		loggingMutex.Lock()
		fmt.Println("")
		fmt.Println("##############################################")
		fmt.Println("#     Backend has finished executing!!!      #")
		fmt.Println("##############################################")
		fmt.Println("")
		loggingMutex.Unlock()
		time.Sleep(1 * time.Hour)
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

var logInfoChan chan string

func handleLogging() {
	for {
		var singleMessage string
		singleMessage = <-logInfoChan

		for i := 0; i < 10000; i++ {
			select {
			case anotherMessage, doesExist := <-logInfoChan:
				if doesExist {
					singleMessage += "\n" + anotherMessage
				}
			default:
				break
			}
		}
		loggingMutex.Lock()
		color.White(singleMessage)
		loggingMutex.Unlock()
	}
}

func logInfo(arrayOfStrings ...string) {
	var message string
	for i := range arrayOfStrings {
		if i > 0 {
			message += " "
		}
		message += arrayOfStrings[i]
	}
	go func() {
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		logInfoChan <- string(t + " " + message)
	}()
}

func checkError(message string, err error) bool {
	var errorFound bool
	if err != nil {
		errorFound = true
		go func() {
			t := time.Now().UTC().Format("2006-01-02 15:04:05")
			err := err.Error() // convert err to string
			loggingMutex.Lock()
			color.Red(t + " " + message + err)
			loggingMutex.Unlock()
		}()
	} else {
		errorFound = false
	}
	return errorFound
}

func logDebug(arrayOfStrings ...string) {
	if debugModeBool == true {
		var message string
		for i := range arrayOfStrings {
			if i > 0 {
				message += " "
			}
			message += arrayOfStrings[i]
		}
		t := time.Now().UTC().Format("2006-01-02 15:04:05")
		loggingMutex.Lock()
		color.Cyan(t + " " + message)
		loggingMutex.Unlock()
	}
}

func growPopulation(currentPopulation float64, foodSupply int32) float64 {
	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	// population if foodSupply == 0
	referencePopulation := float64(10000)

	// maximum % change allowed in single turn (.0005 is standard)
	maxPercentChange := .005

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

func handleInputParameters() {
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
	//   otherwise parse filename to determine if it is a 'blob' file
	//   and import that file if that is the case
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
	//   otherwise parse filename to determine if it is a 'json' file
	//   and import that file if that is the case
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
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// planettype
const (
	TERRAN int = iota
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
	SCOUT int = iota
	COLONY
	CARRIER
	BATTLESHIP
	TITAN
)

// solarsystemtype
const (
	BLUEGIANT int = iota
	YELLOWGIANT
	REDGIANT
	WHITEDWARF
	NEUTRONSTAR
	BLACKHOLE
)

// spaceshipstatus
const (
	OPERATIONAL int = iota
	LIGHTLYDAMAGED
	HEAVILYDAMAGED
	SEVERELYDAMAGED
	HANGINGBYATHREAD
)

// galaxytype
const (
	ELLIPTICAL int = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////
// global variables that are used everywhere
////////////////////////////////////////////////////////////

var (
	galaxyName         string
	inputFilename      string
	outputFilename     string
	googleAudience     string
	authPassword       string
	staticFrontend     string
	frontendBool       bool
	backendBool        bool
	createGalaxyBool   bool
	debugModeBool      bool
	singleThreadedBool bool
)

const (
	constChallenge     = 1
	constGoogleAuth    = 3
	constClientSession = 25
)

var (
	clientBlacklist         map[string]*FirewallClientValues
	clientBlacklistMutex    sync.RWMutex
	clientSessionCount      map[string]uint16
	clientSessionCountMutex sync.RWMutex
)

// FirewallClientValues defines values for firewalling
type FirewallClientValues struct {
	sync.RWMutex
	failedChallengeCount    uint16
	failedChallengeTime     int64
	failedGoogleAuthCount   uint16
	failedGoogleAuthTime    int64
	failedSessionLimitCount uint16
	failedSessionLimitTime  int64
}

var (
	galaxyInfo map[string]*GalaxyInfoStruct
)

// GalaxyInfoStruct defines values for galaxyInfo
type GalaxyInfoStruct struct {
	sync.RWMutex
	ClientIDs []string
	Frontends []string
	Backends  []string
	GlobalX   int64
	GlobalY   int64
}

var (
	numberOfCores  int
	loggingMutex   sync.Mutex
	waitForLogging sync.WaitGroup
)

////////////////////////////////////////////////////////////
// global variables used primarily in 'backend'
////////////////////////////////////////////////////////////

var (
	randomSeed              *rand.Rand
	currentGalaxy           Galaxy
	futureGalaxy            Galaxy
	partitionedSolarSystems []sync.RWMutex
)

var (
	waitForCores sync.WaitGroup
	systemMutex  []sync.RWMutex
	systemIndex  map[string]int
)

////////////////////////////////////////////////////////////
// global variables specific to 'backend_generategalaxy'
////////////////////////////////////////////////////////////

const (
	maxNumberOfPlayers            = 10000
	numSystemsPerPlayer           = 100
	totalNumSystems               = maxNumberOfPlayers * numSystemsPerPlayer
	lightyear                     = fakeToRealMultiplier
	minimumDistanceBetweenSystems = lightyear
	galaxyRadius                  = math.MaxInt64
	realGalaxyRadius              = 125000
	fakeToRealMultiplier          = galaxyRadius / realGalaxyRadius
	oneMillion                    = 1000000
	oneBillion                    = 1000000000
	oneTrillion                   = 1000000000000
)

var (
	rtree        *rtreego.Rtree
	rtreeObjects []rtreego.Spatial
	badSystems   [][]int
)

const (
	numDimensions   = 2
	minBranchFactor = 2
	maxBranchFactor = 2
)

// SystemBox ...
type SystemBox struct {
	location rtreego.Point
	index    int
}

// Bounds ...
func (sb *SystemBox) Bounds() *rtreego.Rect {
	tol := 0.01
	return sb.location.ToRect(tol)
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
