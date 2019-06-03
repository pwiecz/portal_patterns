package main

import "flag"
import "fmt"
import "log"
import "math"

import "os"
import "strings"

import "runtime/pprof"

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to this file")
	cobwebCmd := flag.NewFlagSet("cobweb", flag.ExitOnError)
	cobwebCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s cobweb <portals.json>\n", os.Args[0])
	}
	threeCornersCmd := flag.NewFlagSet("three_corners", flag.ExitOnError)
	threeCornersCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s three_corners <portals1.json> <portals2.json> <portals3.json>\n", os.Args[0])
	}
	herringboneCmd := flag.NewFlagSet("herringbone", flag.ExitOnError)
	herringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s herringbone <portals.json>\n", os.Args[0])
	}
	doubleHerringboneCmd := flag.NewFlagSet("double_herringbone", flag.ExitOnError)
	doubleHerringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s double_herringbone <portals.json>\n", os.Args[0])
	}
	homogeneousCmd := flag.NewFlagSet("homogeneous", flag.ExitOnError)
	homogeneousMaxDepth := homogeneousCmd.Int("max_depth", 6, "don't return homogenous fields with depth larger than max_depth")
	homogeneousLargeTriangles := homogeneousCmd.Bool("pretty", false, "try to split the top triangle into large regular triangles (slow)")
	homogeneousLargestArea := homogeneousCmd.Bool("largest_area", false, "pick the top triangle having the largest possible area")
	homogeneousSmallestArea := homogeneousCmd.Bool("smallest_area", false, "pick the top triangle having the smallest possible area")
	homogeneousCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s homogeneous [--max_depth=<n>] [--pretty] [--largest_area|--smallest_area] <portals.json>\n", os.Args[0])
		homogeneousCmd.PrintDefaults()
	}

	flag.Usage = func() {
		fmt.Println("Usage:")
		cobwebCmd.Usage()
		threeCornersCmd.Usage()
		herringboneCmd.Usage()
		doubleHerringboneCmd.Usage()
		homogeneousCmd.Usage()
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
	switch flag.Args()[0] {
	case "cobweb":
		cobwebCmd.Parse(flag.Args()[1:])
		fileArgs := cobwebCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("cobweb command requires exactly one file argument")
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[0], err)
		}

		result := LargestCobweb(portals)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		portalList := []Portal{result[1], result[0]}
		for _, portal := range result[2:] {
			portalList = append(portalList, portal, portalList[len(portalList)-2])
		}
		fmt.Printf("\n[%s]\n", polylineFromPortalList(portalList))
	case "herringbone":
		herringboneCmd.Parse(flag.Args()[1:])
		fileArgs := herringboneCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("herringbone command requires exactly one file argument")
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[0], err)
		}
		b0, b1, result := LargestHerringbone(portals)
		fmt.Printf("Base (%s) (%s)\n", b0.Name, b1.Name)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		portalList := []Portal{b0, b1}
		atIndex := 1
		for _, portal := range result {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		fmt.Printf("\n[%s]\n", polylineFromPortalList(portalList))
	case "double_herringbone":
		doubleHerringboneCmd.Parse(flag.Args()[1:])
		fileArgs := doubleHerringboneCmd.Args()
		if len(fileArgs) != 1 {
			log.Fatalln("double_herringbone command requires exactly one file argument")
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[0], err)
		}
		b0, b1, result0, result1 := LargestDoubleHerringbone(portals)
		fmt.Printf("Base (%s) (%s)\n", b0.Name, b1.Name)
		fmt.Println("First part:")
		for i, portal := range result0 {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		fmt.Println("Second part:")
		for i, portal := range result1 {
			fmt.Printf("%d: %s\n", i, portal.Name)

		}
		portalList := []Portal{b0, b1}
		atIndex := 1
		for _, portal := range result0 {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		for _, portal := range result1 {
			portalList = append(portalList, portal, portalList[atIndex])
			atIndex = 1 - atIndex
		}
		fmt.Printf("\n[%s]\n", polylineFromPortalList(portalList))
	case "three_corners":
		threeCornersCmd.Parse(flag.Args()[1:])
		fileArgs := threeCornersCmd.Args()
		if len(fileArgs) != 3 {
			log.Fatalln("three_corners command requires exactly three file argument")
		}
		portals1, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[0], err)
		}
		portals2, err := ParseJSONFile(fileArgs[1])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[1], err)
		}
		portals3, err := ParseJSONFile(fileArgs[2])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[3], err)
		}
		if len(portals1)+len(portals2)+len(portals3) >= math.MaxUint16-1 {
			log.Fatalln("Too many portals")
		}
		result := LargestThreeCorner(portals1, portals2, portals3)
		for i, indexedPortal := range result {
			fmt.Printf("%d: %s\n", i, indexedPortal.Portal.Name)
		}
		indexedPortalList := []indexedPortal{result[0], result[1]}
		lastIndexPortal := [3]indexedPortal{result[0], result[1], {}}
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
		portalList := make([]Portal, 0, len(indexedPortalList))
		for _, indexedPortal := range indexedPortalList {
			portalList = append(portalList, indexedPortal.Portal)
		}
		fmt.Printf("\n[%s]\n", polylineFromPortalList(portalList))
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
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			log.Fatalf("Could not parse JSON file %s : %v\n", fileArgs[0], err)
		}
		var result []Portal
		var depth uint16
		var topLevelScorer topLevelTriangleScorer = arbitraryScorer{}
		var scorer homogeneousScorer
		if *homogeneousLargeTriangles {
			if *homogeneousMaxDepth > 7 {
				log.Fatalln("if --pretty is specified --max_depth must be at most 7")
			}
			largeTrianglesScorer := newThickTrianglesScorer(len(portals))
			scorer = largeTrianglesScorer
			topLevelScorer = largeTrianglesScorer
		}
		if *homogeneousLargestArea {
			topLevelScorer = largestTriangleScorer{}
		} else if *homogeneousSmallestArea {
			topLevelScorer = smallestTriangleScorer{}
		}
		if *homogeneousLargeTriangles {
			result, depth = DeepestHomogeneous2(portals, *homogeneousMaxDepth, scorer, topLevelScorer)
		} else {
			result, depth = DeepestHomogeneous(portals, *homogeneousMaxDepth, topLevelScorer)
		}
		fmt.Printf("Depth: %d\n", depth)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		polylines := []string{polylineFromPortalList([]Portal{result[0], result[1], result[2], result[0]})}
		polylines, _ = appendHomogeneousPolylines(result[0], result[1], result[2], uint16(depth), polylines, result[3:])
		fmt.Printf("\n[%s]\n", strings.Join(polylines, ","))
	default:
		log.Fatalf("Unknown command: \"%s\"\n", flag.Args()[0])
	}
}
