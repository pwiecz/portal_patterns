package main

import "flag"
import "fmt"
import "io"
import "log"

import "github.com/pwiecz/portal_patterns/lib"

type homogeneousCmd struct {
	flags         *flag.FlagSet
	maxDepth      *int
	pretty        *bool
	largestArea   *bool
	smallestArea  *bool
	perfect       *bool
	cornerPortals *portalsValue
}

func NewHomogeneousCmd() homogeneousCmd {
	flags := flag.NewFlagSet("homogeneous", flag.ExitOnError)
	cmd := homogeneousCmd{
		flags:         flags,
		maxDepth:      flags.Int("max_depth", 6, "don't return homogenous fields with depth larger than max_depth"),
		pretty:        flags.Bool("pretty", false, "try to split the top triangle into large regular web of triangles (slow)"),
		largestArea:   flags.Bool("largest_area", false, "pick the top triangle having the largest possible area"),
		smallestArea:  flags.Bool("smallest_area", false, "pick the top triangle having the smallest possible area"),
		perfect:       flags.Bool("perfect", false, "consider only perfect homogeneous fields (those that use all the portals inside the top level triangle)"),
		cornerPortals: &portalsValue{},
	}
	flags.Var(cmd.cornerPortals, "corner_portal", "fix corner portal of the homogeneous field")
	return cmd
}

func (h *homogeneousCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s homogeneous [-max_depth=<n>] [-pretty] [-largest_area|-smallest_area] [-corner_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
	h.flags.PrintDefaults()
}

func (h *homogeneousCmd) Run(args []string, output io.Writer, progressFunc func(int, int)) {
	h.flags.Parse(args)
	if *h.maxDepth < 1 {
		log.Fatalln("-max_depth must by at least 1")
	}
	if *h.largestArea && *h.smallestArea {
		log.Fatalln("-largest_area and -smallest_area cannot be both specified at the same time")
	}
	fileArgs := h.flags.Args()
	if len(fileArgs) != 1 {
		log.Fatalln("homogeneous command requires exactly one file argument")
	}
	portals, err := lib.ParseFile(fileArgs[0])
	if err != nil {
		log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
	}
	fmt.Printf("Read %d portals\n", len(portals))
	if len(h.cornerPortals.Portals) > 3 {
		log.Fatalf("homogeneous command accepts at most three corner portals - %d specified", len(h.cornerPortals.Portals))
	}
	cornerPortalIndices := portalsToIndices(*h.cornerPortals, portals)
	if *h.pretty {
		if *h.maxDepth > 7 {
			log.Fatalln("if -pretty is specified -max_depth must be at most 7")
		}
	}
	options := []lib.HomogeneousOption{
		lib.HomogeneousProgressFunc(progressFunc),
		lib.HomogeneousMaxDepth(*h.maxDepth),
		lib.HomogeneousFixedCornerIndices(cornerPortalIndices),
	}
	if *h.largestArea {
		options = append(options, lib.HomogeneousLargestArea{})
	} else if *h.smallestArea {
		options = append(options, lib.HomogeneousSmallestArea{})
	} else if *h.pretty {
		options = append(options, lib.HomogeneousLargestArea{})		
	}
	options = append(options, lib.HomogeneousPerfect(*h.perfect))

	var result []lib.Portal
	var depth uint16
	if *h.pretty {
		result, depth = lib.DeepestHomogeneous2(portals, options...)
	} else {
		result, depth = lib.DeepestHomogeneous(portals, options...)
	}
	fmt.Fprintf(output, "\nDepth: %d\n", depth)
	for i, portal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	drawTools := lib.HomogeneousDrawToolsString(depth, result)
	fmt.Fprintf(output, "\n%s\n", drawTools)
}
