package main

import "flag"
import "fmt"
import "log"
import "runtime"

import "github.com/pwiecz/portal_patterns/lib"

type flipFieldCmd struct {
	flags              *flag.FlagSet
	numBackbonePortals *numberLimitValue
}

func NewFlipFieldCmd() flipFieldCmd {
	flags := flag.NewFlagSet("flip_field", flag.ExitOnError)
	cmd := flipFieldCmd{
		flags:              flags,
		numBackbonePortals: &numberLimitValue{
			Value: 16,
			Exactly: true,
		},
	}
	flags.Var(cmd.numBackbonePortals, "num_backbone_portals", "limit of number of portals in the \"backbone\" of the field. May be a number of have a format of \"<=number\"")
	return cmd
}

func (f *flipFieldCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s flip_field [-num_backbone_portals=[<=]<number>]\n", fileBase)
	f.flags.PrintDefaults()
}

func (f *flipFieldCmd) Run(args []string, numWorkers int, progressFunc func(int, int)) {
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
	numPortalLimit := lib.LESS_EQUAL
	if f.numBackbonePortals.Exactly {
		numPortalLimit = lib.EQUAL
	} else {
		numPortalLimit = lib.LESS_EQUAL
	}
	numFlipFieldWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > 0 {
		numFlipFieldWorkers = numWorkers
	}
	backbone, rest := lib.LargestFlipField(portals, f.numBackbonePortals.Value, numPortalLimit, numFlipFieldWorkers, progressFunc)
	fmt.Printf("\nNum backbone portals: %d, num flip portals: %d, num fields: %d\nBackbone:\n",
		len(backbone), len(rest), len(rest)*(2*len(backbone)-1))
	for i, portal := range backbone {
		fmt.Printf("%d: %s\n", i, portal.Name)
	}
	fmt.Printf("\n[%s,%s]\n", lib.PolylineFromPortalList(backbone), lib.MarkersFromPortalList(rest))

}