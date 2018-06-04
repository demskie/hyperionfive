package main

import (
	"fmt"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/backend"
	"github.com/demskie/hyperionfive/frontend"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/logging"
)

func main() {
	log := logging.New()
	args := arguments.Get(log)
	log.Info(fmt.Sprintf("(hyperionfive) using %v threads",
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
