package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/pwiecz/portal_patterns/lib"
)

type threeCornersCmd struct {
	flags *flag.FlagSet
}

func NewThreeCornersCmd() threeCornersCmd {
	flags := flag.NewFlagSet("three_corners", flag.ExitOnError)
	cmd := threeCornersCmd{
		flags: flags,
	}
	return cmd
}

func (t *threeCornersCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s three_corners <portals1_file> <portals2_file> <portals3_file>\n", fileBase)
	t.flags.PrintDefaults()
}

func (t *threeCornersCmd) Run(args []string, output io.Writer, progressFunc func(int, int)) {
	t.flags.Parse(args)
	fileArgs := t.flags.Args()
	if len(fileArgs) != 3 {
		log.Fatalln("three_corners command requires exactly three file argument")
	}
	portals1, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals(1)\n", len(portals1))
	portals2, err := lib.ParseFile(fileArgs[1])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[1], err)
	}
	fmt.Printf("Read %d portals(2)\n", len(portals2))
	portals3, err := lib.ParseFile(fileArgs[2])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[3], err)
	}
	fmt.Printf("Read %d portals(3)\n", len(portals3))
	if len(portals1)+len(portals2)+len(portals3) >= math.MaxUint16-1 {
		log.Fatalln("Too many portals")
	}

	result := lib.LargestThreeCorner(portals1, portals2, portals3, progressFunc)
	fmt.Fprintln(output, "")
	for i, indexedPortal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, indexedPortal.Portal.Name)
	}
	fmt.Fprintf(output, "\n%s\n", lib.ThreeCornersDrawToolsString(result))
}
