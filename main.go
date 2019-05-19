package main

import "flag"
import "fmt"
import "log"
import "math"
import "os"
import "strings"

import "runtime/pprof"

func main() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	cobwebCmd := flag.NewFlagSet("cobweb", flag.ExitOnError)
	threeCornersCmd := flag.NewFlagSet("three_corners", flag.ExitOnError)
	herringboneCmd := flag.NewFlagSet("herringbone", flag.ExitOnError)
	doubleHerringboneCmd := flag.NewFlagSet("double_herringbone", flag.ExitOnError)
	homogeneousCmd := flag.NewFlagSet("homogeneous", flag.ExitOnError)
	homogeneousMaxDepth := homogeneousCmd.Int("max_depth", 6, "Don't return homogenous fields with depth larger than max_depth")
	homogeneousTrianglulationStrategy := homogeneousCmd.String("triangulation_strategy", "arbitrary", "{arbitrary|avoid_thin_triangles|maximize_min_triangle_height|minimize_triangle_height_variance} - strategy of choosing middle points.")
	homogeneousTopLevelStrategy := homogeneousCmd.String("top_level_strategy", "triangulation", "{triangulation|largest|smallest} - strategy of choosing hightest level triangle")

	if len(os.Args) == 0 {
		fmt.Println("Usage:\n" +
			"  " + os.Args[0] + " cobweb <portals.json>\n" +
			"  " + os.Args[0] + " three_corners <portals1.json> <portals2.json> <portals3.json>\n" +
			"  " + os.Args[0] + " herringbone <portals.json>\n" +
			"  " + os.Args[0] + " double_herringbone <portals.json>\n" +
			"  " + os.Args[0] + " homogeneous <portals.json>")
		return
	}

	/*	portals, err := ParseJSONFile(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", os.Args[2], err)
			os.Exit(1)
		}*/
	if os.Args[1] == "cobweb" {
		cobwebCmd.Parse(os.Args[2:])
		fileArgs := cobwebCmd.Args()
		if len(fileArgs) != 1 {
			fmt.Fprintln(os.Stderr, "cobweb command requires exactly one file argument")
			os.Exit(1)
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[0], err)
			os.Exit(1)
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
	} else if os.Args[1] == "herringbone" {
		herringboneCmd.Parse(os.Args[2:])
		fileArgs := herringboneCmd.Args()
		if len(fileArgs) != 1 {
			fmt.Fprintln(os.Stderr, "herringbone command exactly one file argument")
			os.Exit(1)
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[0], err)
			os.Exit(1)
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
	} else if os.Args[1] == "double_herringbone" {
		doubleHerringboneCmd.Parse(os.Args[2:])
		fileArgs := doubleHerringboneCmd.Args()
		if len(fileArgs) != 1 {
			fmt.Fprintln(os.Stderr, "double_herringbone command exactly one file argument")
			os.Exit(1)
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[0], err)
			os.Exit(1)
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
	} else if os.Args[1] == "three_corners" {
		threeCornersCmd.Parse(os.Args[2:])
		fileArgs := threeCornersCmd.Args()
		if len(fileArgs) != 3 {
			fmt.Fprintln(os.Stderr, "double_herringbone command exactly three file argument")
			os.Exit(1)
		}
		portals1, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[0], err)
			os.Exit(1)
		}
		portals2, err := ParseJSONFile(fileArgs[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[1], err)
			os.Exit(1)
		}
		portals3, err := ParseJSONFile(fileArgs[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[3], err)
			os.Exit(1)
		}
		if len(portals1)+len(portals2)+len(portals3) >= math.MaxUint16-1 {
			fmt.Fprintf(os.Stderr, "Too many portals")
			os.Exit(1)
		}
		result := LargestThreeCorner(portals1, portals2, portals3)
		for i, indexedPortal := range result {
			fmt.Printf("%d: %s\n", i, indexedPortal.Portal.Name)
		}
		indexedPortalList := []indexedPortal{result[0], result[1]}
		lastIndexPortal := [3]indexedPortal{result[0], result[1], indexedPortal{}}
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
	} else if os.Args[1] == "homogeneous" || os.Args[1] == "homogenous" {
		homogeneousCmd.Parse(os.Args[2:])
		if *homogeneousMaxDepth < 1 {
			fmt.Fprintln(os.Stderr, "max_depth must by at least 1")
			os.Exit(1)
		}
		fileArgs := homogeneousCmd.Args()
		if len(fileArgs) != 1 {
			fmt.Fprintln(os.Stderr, "herringbone command exactly one file argument")
			os.Exit(1)
		}
		portals, err := ParseJSONFile(fileArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", fileArgs[0], err)
			os.Exit(1)
		}
		var result []Portal
		var depth uint16
		if *homogeneousTrianglulationStrategy == "arbitrary" || *homogeneousTrianglulationStrategy == "avoid_thin_triangles" {
			var scorer homogeneousScorer
			var topLevelScorer homogeneousTopLevelScorer
			switch *homogeneousTrianglulationStrategy {
			case "arbitrary":
				scorer = arbitraryScorer{}
				topLevelScorer = arbitraryScorer{}
			case "avoid_thin_triangles":
				s := newAvoidThinTrianglesScorer(len(portals))
				scorer = s
				topLevelScorer = s
			}
			switch *homogeneousTopLevelStrategy {
			case "triangulation":
			case "largest":
				topLevelScorer = largestTriangleTopLevelScorer{}
			case "smallest":
				topLevelScorer = smallestTriangleTopLevelScorer{}
			default:
				fmt.Fprintf(os.Stderr, "Unknown top_level_strategy %f\n", *homogeneousTopLevelStrategy)
				os.Exit(1)
			}
			result, depth = DeepestHomogeneous(portals, *homogeneousMaxDepth, scorer, topLevelScorer)
		} else if *homogeneousTrianglulationStrategy == "maximize_min_triangle_height" || *homogeneousTrianglulationStrategy == "minimize_triangle_height_variance" {
			if *homogeneousMaxDepth > 6 {
				fmt.Fprintf(os.Stderr, "%s strategy support max_depth at most 6\n", *homogeneousTrianglulationStrategy)
				os.Exit(1)
			}
			var scorer homogeneous2Scorer
			var topLevelScorer homogeneous2TopLevelScorer
			switch *homogeneousTrianglulationStrategy {
			case "maximize_min_triangle_height":
				s := newAvoidThinTriangles2Scorer(len(portals))
				scorer = s
				topLevelScorer = s
			case "minimize_triangle_height_variance":
				s := newMinHeightVarianceScorer(len(portals))
				scorer = s
				topLevelScorer = s
			}
			switch *homogeneousTopLevelStrategy {
			case "triangulation":
			case "largest":
				topLevelScorer = largestTriangleTopLevelScorer{}
			case "smallest":
				topLevelScorer = smallestTriangleTopLevelScorer{}
			default:
				fmt.Fprintf(os.Stderr, "Unknown top_level_strategy %f\n", *homogeneousTopLevelStrategy)
				os.Exit(1)
			}
			result, depth = DeepestHomogeneous2(portals, 6, scorer, topLevelScorer)
		}
		fmt.Printf("Depth: %d\n", depth+1)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		polylines := []string{polylineFromPortalList([]Portal{result[0], result[1], result[2], result[0]})}
		polylines, _ = appendHomogeneousPolylines(result[0], result[1], result[2], uint16(depth), polylines, result[3:])
		fmt.Printf("\n[%s]\n", strings.Join(polylines, ","))
	} else {
		fmt.Fprintf(os.Stderr, "Unknown command: \"%s\"\n", os.Args[1])
		os.Exit(1)
	}
}
