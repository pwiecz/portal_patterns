package main

import (
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/pwiecz/portal_patterns/lib"
)

type doubleHerringboneCmd struct {
	flags       *flag.FlagSet
	basePortals *portalsValue
}

func NewDoubleHerringboneCmd() doubleHerringboneCmd {
	flags := flag.NewFlagSet("double_herringbone", flag.ExitOnError)
	cmd := doubleHerringboneCmd{
		flags:       flags,
		basePortals: &portalsValue{},
	}
	flags.Var(cmd.basePortals, "base_portal", "fix a base portal of the double herringbone field")
	return cmd
}

func (d *doubleHerringboneCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s double_herringbone [-base_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
	d.flags.PrintDefaults()
}

func (d *doubleHerringboneCmd) Run(args []string, output io.Writer, numWorkers int, progressFunc func(int, int)) {
	d.flags.Parse(args)
	fileArgs := d.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("double_herringbone command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))
	if len(*d.basePortals) > 2 {
		log.Fatalf("double_herringbone command accepts at most two base portals - %d specified", len(*d.basePortals))
	}
	basePortalIndices := portalsToIndices(*d.basePortals, portals)

	b0, b1, result0, result1 := lib.LargestDoubleHerringbone(portals, basePortalIndices, numWorkers, progressFunc)
	fmt.Fprintf(output, "\nBase (%s) (%s)\n", b0.Name, b1.Name)
	fmt.Fprintln(output, "First part:")
	for i, portal := range result0 {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	fmt.Fprintln(output, "Second part:")
	for i, portal := range result1 {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)

	}
	fmt.Fprintf(output, "\n%s\n", lib.DoubleHerringboneDrawToolsString(b0, b1, result0, result1))
}
