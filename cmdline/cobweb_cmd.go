package main

import (
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/pwiecz/portal_patterns/lib"
)

type cobwebCmd struct {
	flags         *flag.FlagSet
	cornerPortals *portalsValue
}

func NewCobwebCmd() cobwebCmd {
	flags := flag.NewFlagSet("cobweb", flag.ExitOnError)
	cmd := cobwebCmd{
		flags:         flags,
		cornerPortals: &portalsValue{},
	}
	flags.Var(cmd.cornerPortals, "corner_portal", "fix corner portal of the cobweb field")
	return cmd
}

func (c *cobwebCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s cobweb [-corner_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
	c.flags.PrintDefaults()
}

func (c *cobwebCmd) Run(args []string, output io.Writer, progressFunc func(int, int)) {
	c.flags.Parse(args)
	fileArgs := c.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("cobweb command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))
	if len(*c.cornerPortals) > 3 {
		log.Fatalf("cobweb command accepts at most three corner portals - %d specified", len(*c.cornerPortals))
	}
	cornerPortalIndices := portalsToIndices(*c.cornerPortals, portals)

	result := lib.LargestCobweb(portals, cornerPortalIndices, progressFunc)

	fmt.Fprintln(output, "")
	for i, portal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	fmt.Fprintf(output, "\n%s\n", lib.CobwebDrawToolsString(result))
}
