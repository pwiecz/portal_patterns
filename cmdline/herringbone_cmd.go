package main

import (
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/pwiecz/portal_patterns/lib"
)

type herringboneCmd struct {
	flags       *flag.FlagSet
	basePortals *portalsValue
}

func NewHerringboneCmd() herringboneCmd {
	flags := flag.NewFlagSet("herringbone", flag.ExitOnError)
	cmd := herringboneCmd{
		flags:       flags,
		basePortals: &portalsValue{},
	}
	flags.Var(cmd.basePortals, "base_portal", "fix a base portal of the herringbone field")
	return cmd
}

func (h *herringboneCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s herringbone [-base_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
	h.flags.PrintDefaults()
}

func (h *herringboneCmd) Run(args []string, output io.Writer, numWorkers int, progressFunc func(int, int)) {
	h.flags.Parse(args)
	fileArgs := h.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("herringbone command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))
	if len(*h.basePortals) > 2 {
		log.Fatalf("herringbone command accepts at most two corner portals - %d specified", len(*h.basePortals))
	}
	basePortalIndices := portalsToIndices(*h.basePortals, portals)

	b0, b1, result := lib.LargestHerringbone(portals, basePortalIndices, numWorkers, progressFunc)
	fmt.Fprintf(output, "\nBase (%s) (%s)\n", b0.Name, b1.Name)
	for i, portal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	fmt.Fprintf(output, "\n%s\n", lib.HerringboneDrawToolsString(b0, b1, result))
}
