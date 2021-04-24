package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/pwiecz/portal_patterns/lib"
)

type homogeneousCmd struct {
	flags           *flag.FlagSet
	maxDepth        *int
	pretty          *bool
	largestArea     *bool
	smallestArea    *bool
	mostEquilateral *bool
	random          *bool
	pure            *bool
	cornerPortals   *portalsValue
}

func NewHomogeneousCmd() homogeneousCmd {
	flags := flag.NewFlagSet("homogeneous", flag.ExitOnError)
	cmd := homogeneousCmd{
		flags:           flags,
		maxDepth:        flags.Int("max_depth", 6, "don't return homogenous fields with depth larger than max_depth"),
		pretty:          flags.Bool("pretty", false, "try to split the top triangle into large regular web of triangles (slow)"),
		largestArea:     flags.Bool("largest_area", false, "pick the top triangle having the largest possible area"),
		smallestArea:    flags.Bool("smallest_area", false, "pick the top triangle having the smallest possible area (default)"),
		mostEquilateral: flags.Bool("most_equilateral", false, "pick the top triangle being the most equilateral"),
		random:          flags.Bool("random", false, "pick a random top triangle"),
		pure:            flags.Bool("pure", false, "consider only pure homogeneous fields (those that use all the portals inside the top level triangle)"),
		cornerPortals:   &portalsValue{},
	}
	flags.Var(cmd.cornerPortals, "corner_portal", "fix corner portal of the homogeneous field")
	return cmd
}

func (h *homogeneousCmd) Usage(fileBase string) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s homogeneous [-max_depth=<n>] [-pretty] [-largest_area|-smallest_area|-most_equilateral|-random] [-pure] [-corner_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
	h.flags.PrintDefaults()
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (h *homogeneousCmd) Run(args []string, output io.Writer, numWorkers int, progressFunc func(int, int)) {
	h.flags.Parse(args)
	if *h.maxDepth < 1 {
		log.Fatalln("-max_depth must by at least 1")
	}
	if btoi(*h.largestArea)+btoi(*h.smallestArea)+btoi(*h.mostEquilateral)+btoi(*h.random) > 1 {
		log.Fatalln("only one of -largest_area -smallest_area -most_equilateral -random can be specified at the same time")
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
	if len(*h.cornerPortals) > 3 {
		log.Fatalf("homogeneous command accepts at most three corner portals - %d specified", len(*h.cornerPortals))
	}
	cornerPortalIndices := portalsToIndices(*h.cornerPortals, portals)
	options := []lib.HomogeneousOption{
		lib.HomogeneousNumWorkers(numWorkers),
		lib.HomogeneousProgressFunc(progressFunc),
		lib.HomogeneousMaxDepth(*h.maxDepth),
		lib.HomogeneousFixedCornerIndices(cornerPortalIndices),
	}
	// check for pretty before setting top level scorer, as pretty overwrites the top level scorer
	if *h.pretty {
		if !*h.pure && *h.maxDepth > 7 {
			log.Fatalln("if -pretty is specified and -pure is not then -max_depth must be at most 7")
		}
		options = append(options, lib.HomogeneousSpreadAround{})
	}
	if *h.largestArea {
		options = append(options, lib.HomogeneousLargestArea{})
	} else if *h.smallestArea {
		options = append(options, lib.HomogeneousSmallestArea{})
	} else if *h.mostEquilateral {
		options = append(options, lib.HomogeneousMostEquilateralTriangle{})
	} else if *h.random {
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		options = append(options, lib.HomogeneousRandom{Rand: rand})
	}
	options = append(options, lib.HomogeneousPure(*h.pure))

	result, depth := lib.DeepestHomogeneous(portals, options...)

	fmt.Fprintf(output, "\nDepth: %d\n", depth)
	for i, portal := range result {
		fmt.Fprintf(output, "%d: %s\n", i, portal.Name)
	}
	drawTools := lib.HomogeneousDrawToolsString(depth, result)
	fmt.Fprintf(output, "\n%s\n", drawTools)
}
