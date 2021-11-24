package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"runtime"

	"github.com/pwiecz/portal_patterns/lib"
)

type flipFieldCmd struct {
	flags              *flag.FlagSet
	numBackbonePortals *numberLimitValue
	maxFlipPortals     *int
	simpleBackbone     *bool
}

func NewFlipFieldCmd() flipFieldCmd {
	flags := flag.NewFlagSet("flip_field", flag.ExitOnError)
	cmd := flipFieldCmd{
		flags: flags,
		numBackbonePortals: &numberLimitValue{
			Value:   16,
			Exactly: true,
		},
		maxFlipPortals: flags.Int("max_flip_portals", 0, "if >0 don't try to optimize for number of flip portals above this value"),
		simpleBackbone: flags.Bool("simple_backbone", false, "make all backbone portals linkable from the first backbone portal"),
	}
	flags.Var(cmd.numBackbonePortals, "num_backbone_portals", "limit of number of portals in the \"backbone\" of the field. May be a number of have a format of \"<=number\"")
	return cmd
}

func (f *flipFieldCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s flip_field [-num_backbone_portals=[<=]<number>] [--max_flip_portals=<number>] [--simple_backbone]\n", fileBase)
	f.flags.PrintDefaults()
}

func (f *flipFieldCmd) Run(args []string, numWorkers int, output io.Writer, progressFunc func(int, int)) {
	f.flags.Parse(flag.Args()[1:])
	if f.numBackbonePortals.Value <= 2 {
		log.Fatalln("-num_backbone_portals limit must be at least 2")
	}
	fileArgs := f.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("flip_field command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))
	var numPortalLimit lib.PortalLimit
	if f.numBackbonePortals.Exactly {
		numPortalLimit = lib.EQUAL
	} else {
		numPortalLimit = lib.LESS_EQUAL
	}
	numFlipFieldWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > 0 {
		numFlipFieldWorkers = numWorkers
	}
	options := []lib.FlipFieldOption{
		lib.FlipFieldProgressFunc(progressFunc),
		lib.FlipFieldNumWorkers(numFlipFieldWorkers),
		lib.FlipFieldBackbonePortalLimit{Value: f.numBackbonePortals.Value, LimitType: numPortalLimit},
		lib.FlipFieldMaxFlipPortals(*f.maxFlipPortals),
		lib.FlipFieldSimpleBackbone(*f.simpleBackbone),
	}
	backbone, rest := lib.LargestFlipField(portals, options...)
	fmt.Fprintf(output, "\nNum backbone portals: %d, num flip portals: %d, num fields: %d\nBackbone:\n",
		len(backbone), len(rest), len(rest)*(2*len(backbone)-3))
	for i, portal := range backbone {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	fmt.Fprintf(output, "\n[%s", lib.PolylineFromPortalList(backbone))
	if len(rest) > 0 {
		fmt.Fprintf(output, ",%s", lib.MarkersFromPortalList(rest))
	}
	fmt.Fprintln(output, "]")
}
