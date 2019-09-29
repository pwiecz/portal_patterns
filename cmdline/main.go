package main

import "flag"
import "fmt"
import "log"
import "math"
import "runtime"

import "os"
import "strings"

import "path/filepath"
import "runtime/pprof"

import "github.com/pwiecz/portal_patterns/lib"

func main() {
	fileBase := filepath.Base(os.Args[0])
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to this file")
	numWorkers := flag.Int("num_workers", 0, "if applicable for given algorithm use that many worker threads. If <= 0 use as many as there are CPUs on the machine")
	showProgress := flag.Bool("progress", true, "show progress bar")
	flag.BoolVar(showProgress, "P", true, "show progress bar")
	cobwebCmd := flag.NewFlagSet("cobweb", flag.ExitOnError)
	cobwebCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s cobweb <portals>\n", fileBase)
	}
	threeCornersCmd := flag.NewFlagSet("three_corners", flag.ExitOnError)
	threeCornersCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s three_corners <portals1> <portals2> <portals3>\n", fileBase)
	}
	herringboneCmd := flag.NewFlagSet("herringbone", flag.ExitOnError)
	herringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s herringbone <portals>\n", fileBase)
	}
	doubleHerringboneCmd := flag.NewFlagSet("double_herringbone", flag.ExitOnError)
	doubleHerringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s double_herringbone <portals>\n", fileBase)
	}
	homogeneousCmd := flag.NewFlagSet("homogeneous", flag.ExitOnError)
	homogeneousMaxDepth := homogeneousCmd.Int("max_depth", 6, "don't return homogenous fields with depth larger than max_depth")
	homogeneousLargeTriangles := homogeneousCmd.Bool("pretty", false, "try to split the top triangle into large regular web of triangles (slow)")
	homogeneousLargestArea := homogeneousCmd.Bool("largest_area", false, "pick the top triangle having the largest possible area")
	homogeneousSmallestArea := homogeneousCmd.Bool("smallest_area", false, "pick the top triangle having the smallest possible area")
	homogeneousCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s homogeneous [-max_depth=<n>] [-pretty] [-largest_area|-smallest_area] <portals>\n", fileBase)
		homogeneousCmd.PrintDefaults()
	}
	colinearCmd := flag.NewFlagSet("colinear", flag.ExitOnError)
	colinearCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s colinear <portals>\n", fileBase)
	}

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		cobwebCmd.Usage()
		threeCornersCmd.Usage()
		herringboneCmd.Usage()
		doubleHerringboneCmd.Usage()
		homogeneousCmd.Usage()
		colinearCmd.Usage()
	}
	flag.Parse()
	if len(flag.Args()) <= 1 {
		flag.Usage()
		os.Exit(0)
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	progressFunc := lib.PrintProgressBar
	if !*showProgress {
		progressFunc = func(int, int) {}
	}
	switch flag.Args()[0] {
	case "cobweb":
		cobwebCmd.Parse(flag.Args()[1:])
		fileArgs := cobwebCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("cobweb command requires exactly one file argument")
		}
		portals, err := lib.ParseFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
		}

		result := lib.LargestCobweb(portals, progressFunc)
		fmt.Println("")
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		portalList := []lib.Portal{result[1], result[0]}
		for _, portal := range result[2:] {
			portalList = append(portalList, portal, portalList[len(portalList)-2])
		}
		fmt.Printf("\n[%s]\n", lib.PolylineFromPortalList(portalList))
	case "herringbone":
		herringboneCmd.Parse(flag.Args()[1:])
		fileArgs := herringboneCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("herringbone command requires exactly one file argument")
		}
		portals, err := lib.ParseFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
		}
		numHerringboneWorkers := runtime.GOMAXPROCS(0)
		if *numWorkers > 0 {
			numHerringboneWorkers = *numWorkers
		}
		b0, b1, result := lib.LargestHerringbone(portals, numHerringboneWorkers, progressFunc)
		fmt.Printf("\nBase (%s) (%s)\n", b0.Name, b1.Name)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		portalList := []lib.Portal{b0, b1}
		atIndex := 1
		for _, portal := range result {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		fmt.Printf("\n[%s]\n", lib.PolylineFromPortalList(portalList))
	case "double_herringbone":
		doubleHerringboneCmd.Parse(flag.Args()[1:])
		fileArgs := doubleHerringboneCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("double_herringbone command requires exactly one file argument")
		}
		portals, err := lib.ParseFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
		}
		numHerringboneWorkers := runtime.GOMAXPROCS(0)
		if *numWorkers > 0 {
			numHerringboneWorkers = *numWorkers
		}
		b0, b1, result0, result1 := lib.LargestDoubleHerringbone(portals, numHerringboneWorkers, progressFunc)
		fmt.Printf("\nBase (%s) (%s)\n", b0.Name, b1.Name)
		fmt.Println("First part:")
		for i, portal := range result0 {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		fmt.Println("Second part:")
		for i, portal := range result1 {
			fmt.Printf("%d: %s\n", i, portal.Name)

		}
		portalList := []lib.Portal{b0, b1}
		atIndex := 1
		for _, portal := range result0 {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		for _, portal := range result1 {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		fmt.Printf("\n[%s]\n", lib.PolylineFromPortalList(portalList))
	case "three_corners":
		threeCornersCmd.Parse(flag.Args()[1:])
		fileArgs := threeCornersCmd.Args()
		if len(fileArgs) != 3 {
			log.Fatalln("three_corners command requires exactly three file argument")
		}
		portals1, err := lib.ParseFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
		}
		portals2, err := lib.ParseFile(fileArgs[1])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[1], err)
		}
		portals3, err := lib.ParseFile(fileArgs[2])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[3], err)
		}
		if len(portals1)+len(portals2)+len(portals3) >= math.MaxUint16-1 {
			log.Fatalln("Too many portals")
		}
		result := lib.LargestThreeCorner(portals1, portals2, portals3, progressFunc)
		fmt.Println("")
		for i, indexedPortal := range result {
			fmt.Printf("%d: %s\n", i, indexedPortal.Portal.Name)
		}
		indexedPortalList := []lib.IndexedPortal{result[0], result[1]}
		lastIndexPortal := [3]lib.IndexedPortal{result[0], result[1], {}}
		for _, indexedPortal := range result[2:] {
			lastIndex := indexedPortalList[len(indexedPortalList)-1].Index
			if indexedPortal.Index == lastIndex {
				lastIndex = (lastIndex + 1) % 3
				indexedPortalList = append(indexedPortalList, lastIndexPortal[lastIndex])
			}
			nextIndex := 3 - indexedPortal.Index - lastIndex
			indexedPortalList = append(indexedPortalList, indexedPortal, lastIndexPortal[nextIndex])
			lastIndexPortal[indexedPortal.Index] = indexedPortal
		}
		portalList := make([]lib.Portal, 0, len(indexedPortalList))
		for _, indexedPortal := range indexedPortalList {
			portalList = append(portalList, indexedPortal.Portal)
		}
		fmt.Printf("\n[%s]\n", lib.PolylineFromPortalList(portalList))
	case "homogeneous":
		fallthrough
	case "homogenous":
		homogeneousCmd.Parse(flag.Args()[1:])
		if *homogeneousMaxDepth < 1 {
			log.Fatalln("--max_depth must by at least 1")
		}
		if *homogeneousLargestArea && *homogeneousSmallestArea {
			log.Fatalln("--largest_area and --smallest_area cannot be both specified at the same time")
		}
		fileArgs := homogeneousCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("homogeneous command requires exactly one file argument")
		}
		portals, err := lib.ParseFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse file %s : %v\n", fileArgs[0], err)
		}
		var result []lib.Portal
		var depth uint16
		var topLevelScorer lib.TopLevelTriangleScorer = lib.ArbitraryScorer{}
		var scorer lib.HomogeneousScorer
		if *homogeneousLargeTriangles {
			if *homogeneousMaxDepth > 7 {
				log.Fatalln("if --pretty is specified --max_depth must be at most 7")
			}
			largeTrianglesScorer := lib.NewThickTrianglesScorer(len(portals))
			scorer = largeTrianglesScorer
			topLevelScorer = largeTrianglesScorer
		}
		if *homogeneousLargestArea {
			topLevelScorer = lib.LargestTriangleScorer{}
		} else if *homogeneousSmallestArea {
			topLevelScorer = lib.SmallestTriangleScorer{}
		}

		if *homogeneousLargeTriangles {
			result, depth = lib.DeepestHomogeneous2(portals, *homogeneousMaxDepth, scorer, topLevelScorer, progressFunc)
		} else {
			result, depth = lib.DeepestHomogeneous(portals, *homogeneousMaxDepth, topLevelScorer, progressFunc)
		}
		fmt.Printf("\nDepth: %d\n", depth)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		polylines := []string{lib.PolylineFromPortalList([]lib.Portal{result[0], result[1], result[2], result[0]})}
		polylines, _ = lib.AppendHomogeneousPolylines(result[0], result[1], result[2], uint16(depth), polylines, result[3:])
		fmt.Printf("\n[%s]\n", strings.Join(polylines, ","))
	default:
		log.Fatalf("Unknown command: \"%s\"\n", flag.Args()[0])
	}
}
