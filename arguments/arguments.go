package arguments

import (
	"flag"
	"runtime"
	"strconv"

	"github.com/demskie/logging"
)

// Args ...
type Args struct {
	InputFile       string
	OutputFile      string
	GoogleAudience  string
	AuthPassword    string
	StaticFrontend  string
	LocalInterface  string
	LocalAddress    string
	DebugModeBool   bool
	NumberOfCores   int
	Mode            string
	BackendHostname string
	BackendGalaxyID int32
	BackendGlobalX  int64
	BackendGlobalY  int64
}

// Get ...
func Get(log *logging.Logger) (args *Args) {
	var (
		inputFilePtr          = flag.String("input", "", "specify data source, either json or blob")
		outputFilePtr         = flag.String("output", "", "override export destination, either json or blob")
		googleAudiencePtr     = flag.String("audience", "", "override google oauth audience")
		authPasswordPtr       = flag.String("password", "", "specify password for authentication purposes")
		staticFrontendPtr     = flag.String("staticfrontend", "", "specify static frontend address")
		localInterfacePtr     = flag.String("interface", "", "override local interface to use")
		localAddressPtr       = flag.String("localip", "", "override local ip to use")
		debugModeBoolPtr      = flag.Bool("debug", false, "enable debug logging")
		singleThreadedBoolPtr = flag.Bool("singlethreaded", false, "force singlethreaded behavior")
		enableFrontendBoolPtr = flag.Bool("frontend", false, "enable frontend mode")
		enableBackendBoolPtr  = flag.Bool("backend", false, "enable backend mode")

		backendHostnamePtr = flag.String("backendhostname", "", "specify backend hostname")
		backendGalaxyIDPtr = flag.String("backendgalaxyid", "", "specify backend galaxyID")
		backendGlobalXPtr  = flag.Int64("globalx", 0, "specify backend globalX")
		backendGlobalYPtr  = flag.Int64("globaly", 0, "specify backend globalY")
	)

	flag.Parse()
	args = &Args{}

	args.InputFile = *inputFilePtr

	args.OutputFile = *outputFilePtr

	args.GoogleAudience = *googleAudiencePtr
	if args.GoogleAudience == "" {
		args.GoogleAudience = "771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
	}

	args.AuthPassword = *authPasswordPtr
	if args.AuthPassword == "" {
		log.Info("(arguments) no password provided, using default password")
		args.AuthPassword += "uVXMHpASbDngPe41pUP1"
		args.AuthPassword += "CKwRCU6ZGap6IxQ6OQaM"
		args.AuthPassword += "AtJgeaIhHLzO0IibxE8B"
		args.AuthPassword += "7aYOpAXkC388SRg4G3jO"
		args.AuthPassword += "kkgrBnOOUPxnT6QopsFH"
		args.AuthPassword += "a0F7qFScNxYuCaiRvii3"
		args.AuthPassword += "WvhhNNsFaRupwZc7SXXw"
		args.AuthPassword += "cPk93TWZs0kllqVhCN0G"
		args.AuthPassword += "9oKiFpa0p7Mfwd8CqHiz"
		args.AuthPassword += "3ghjZw6FI17DjdObCXSF"
	}

	args.StaticFrontend = *staticFrontendPtr

	args.LocalInterface = *localInterfacePtr

	args.LocalAddress = *localAddressPtr

	args.DebugModeBool = *debugModeBoolPtr
	if args.DebugModeBool {
		log.EnableDebugMode()
	}

	if *singleThreadedBoolPtr == true {
		args.NumberOfCores = 1
	} else {
		args.NumberOfCores = runtime.NumCPU()
	}

	if *enableFrontendBoolPtr && *enableBackendBoolPtr {
		log.Error("cannot be both frontend and backend")
		log.SuddenExit()
	} else if *enableFrontendBoolPtr {
		args.Mode = "frontend"
	} else if *enableBackendBoolPtr {
		args.Mode = "backend"
	}

	if args.Mode == "backend" {
		args.BackendHostname = *backendHostnamePtr

		val, err := strconv.ParseInt(*backendGalaxyIDPtr, 10, 64)
		if log.CheckError(err, "invalid/missing int32 galaxyNameID") {
			log.SuddenExit()
		}
		args.BackendGalaxyID = int32(val)

		args.BackendGlobalX = *backendGlobalXPtr

		args.BackendGlobalY = *backendGlobalYPtr
	}

	return args
}
