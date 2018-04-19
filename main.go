package main

import (
	"fmt"
	"hyperionfive/backend"
	"hyperionfive/frontend"
	"hyperionfive/intutil"
	"hyperionfive/logging"
)

func main() {
	log := logging.New()
	args := arguments.GetArgs(log)
	log.Info(fName, fmt.Sprintf("using %v threads",
		intutil.GetNumberString(args.NumberOfCores)))

	switch {
	case args.EnableFrontendBool:
		frontend.Start(args, log)
	case args.EnableBackendBool:
		backend.Start(args, log)
	default:
		log.Error("no operating mode was specified")
	}

	log.Error("finished executing")
	log.SuddenExit()
}
