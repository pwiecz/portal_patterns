package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"runtime"

	"github.com/pwiecz/portal_patterns/lib"
)

type droneFlightCmd struct {
	flags        *flag.FlagSet
	useLongJumps *bool
	startPortal  *portalValue
	endPortal    *portalValue
}

func NewDroneFlightCmd() droneFlightCmd {
	flags := flag.NewFlagSet("drone_flight", flag.ExitOnError)
	cmd := droneFlightCmd{
		flags:        flags,
		useLongJumps: flags.Bool("use_long_jumps", true, "when creating the drone flight consider using long jumps that require a key to the target portal"),
		startPortal:  &portalValue{},
		endPortal:    &portalValue{},
	}
	flags.Var(cmd.startPortal, "start_portal", "fix the start portal of the drone flight path")
	flags.Var(cmd.endPortal, "end_portal", "fix the end portal of the drone flight path")
	return cmd
}

func (d *droneFlightCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s drone_flight [--start_portal=<lat>,<lng>] [--end_portal=<lat>,<lng>] [--use_long_jumps]\n", fileBase)
	d.flags.PrintDefaults()
}

func (d *droneFlightCmd) Run(args []string, numWorkers int, output io.Writer, progressFunc func(int, int)) {
	d.flags.Parse(flag.Args()[1:])
	fileArgs := d.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("drone_flight command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))

	numDroneFlightWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > 0 {
		numDroneFlightWorkers = numWorkers
	}

	options := []lib.DroneFlightOption{
		lib.DroneFlightProgressFunc(progressFunc),
		lib.DroneFlightNumWorkers(numDroneFlightWorkers),
		lib.DroneFlightStartPortalIndex(portalToIndex(*d.startPortal, portals)),
		lib.DroneFlightEndPortalIndex(portalToIndex(*d.endPortal, portals)),
		lib.DroneFlightUseLongJumps(*d.useLongJumps),
	}
	result, keysNeeded := lib.LongestDroneFlight(portals, options...)
	distance := result[0].LatLng.Distance(result[len(result)-1].LatLng) * lib.RadiansToMeters
	fmt.Fprintln(output, "")
	fmt.Fprintf(output, "Max flight distance: %fm\n", distance)
	fmt.Fprintf(output, "Keys needed: %d\n", len(keysNeeded))
	for i, portal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	fmt.Fprintf(output, "\n[%s", lib.PolylineFromPortalList(result))
	if len(keysNeeded) > 0 {
		fmt.Fprintf(output, ",%s", lib.MarkersFromPortalList(keysNeeded))
	}
	fmt.Fprintln(output, "]")
}
