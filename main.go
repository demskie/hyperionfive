package main

import (
	"fmt"
	"hyperionfive/arguments"
	"hyperionfive/backend"
	"hyperionfive/frontend"
	"hyperionfive/intutil"
	"hyperionfive/logging"
)

func main() {
	log := logging.New()
	args := arguments.Get(log)
	log.Info(fmt.Sprintf("(main) using %v threads",
		intutil.GetNumberString(args.NumberOfCores)))

	switch args.Mode {
	case "frontend":
		frontend.Start(args, log)
	case "backend":
		backend.Start(args, log)
	default:
		log.Error("no operating mode was specified")
	}

	log.Error("finished executing")
	log.SuddenExit()
}
