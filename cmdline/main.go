package main

import "flag"
import "fmt"
import "log"
import "math"
import "os"
import "runtime"

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
	cobwebCornerPortalsValue := portalsValue{}
	cobwebCmd.Var(&cobwebCornerPortalsValue, "corner_portal", "fix corner portal of the cobweb field")
	cobwebCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s cobweb [--corner_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
		cobwebCmd.PrintDefaults()
	}
	threeCornersCmd := flag.NewFlagSet("three_corners", flag.ExitOnError)
	threeCornersCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s three_corners <portals1_file> <portals2_file> <portals3_file>\n", fileBase)
	}
	herringboneCmd := flag.NewFlagSet("herringbone", flag.ExitOnError)
	herringboneBasePortalsValue := portalsValue{}
	herringboneCmd.Var(&herringboneBasePortalsValue, "base_portal", "fix a base portal of the herringbone field")
	herringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s herringbone [--base_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
		herringboneCmd.PrintDefaults()
	}
	doubleHerringboneCmd := flag.NewFlagSet("double_herringbone", flag.ExitOnError)
	doubleHerringboneBasePortalsValue := portalsValue{}
	doubleHerringboneCmd.Var(&doubleHerringboneBasePortalsValue, "base_portal", "fix a base portal of the double herringbone field")
	doubleHerringboneCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s double_herringbone [--base_portal=<lat>,<lng>]... <portals_file>\n", fileBase)
		doubleHerringboneCmd.PrintDefaults()
	}
	homogeneousCmd := NewHomogeneousCmd()

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		cobwebCmd.Usage()
		threeCornersCmd.Usage()
		herringboneCmd.Usage()
		doubleHerringboneCmd.Usage()
		homogeneousCmd.Usage(fileBase)
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
		if len(cobwebCornerPortalsValue.Portals) > 3 {
			log.Fatalf("cobweb command accepts at most three corner portals - %d specified", len(cobwebCornerPortalsValue.Portals))
		}
		cobwebCornerPortalIndices := portalsToIndices(cobwebCornerPortalsValue, portals)

		result := lib.LargestCobweb(portals, cobwebCornerPortalIndices, progressFunc)
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
		if len(herringboneBasePortalsValue.Portals) > 2 {
			log.Fatalf("herringbone command accepts at most two base portals - %d specified", len(herringboneBasePortalsValue.Portals))
		}
		herringboneBasePortalIndices := portalsToIndices(herringboneBasePortalsValue, portals)
		numHerringboneWorkers := runtime.GOMAXPROCS(0)
		if *numWorkers > 0 {
			numHerringboneWorkers = *numWorkers
		}
		b0, b1, result := lib.LargestHerringbone(portals, herringboneBasePortalIndices, numHerringboneWorkers, progressFunc)
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
		if len(doubleHerringboneBasePortalsValue.Portals) > 2 {
			log.Fatalf("double_herringbone command accepts at most two base portals - %d specified", len(doubleHerringboneBasePortalsValue.Portals))
		}
		doubleHerringboneBasePortalIndices := portalsToIndices(doubleHerringboneBasePortalsValue, portals)
		numHerringboneWorkers := runtime.GOMAXPROCS(0)
		if *numWorkers > 0 {
			numHerringboneWorkers = *numWorkers
		}
		b0, b1, result0, result1 := lib.LargestDoubleHerringbone(portals, doubleHerringboneBasePortalIndices, numHerringboneWorkers, progressFunc)
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
		homogeneousCmd.Run(flag.Args()[1:], progressFunc)
	default:
		log.Fatalf("Unknown command: \"%s\"\n", flag.Args()[0])
	}
}
