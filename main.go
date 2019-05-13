package main

import "fmt"
import "os"
import "strings"

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:\n" +
		 "  " + os.Args[0] + " cobweb <portals.json>\n" + 
		 "  " + os.Args[0] + " three_corners <portals1.json> <portals2.json> <portals3.json>\n" +
		 "  " + os.Args[0] + " herringbone <portals.json>\n" +
		 "  " + os.Args[0] + " double_herringbone <portals.json>\n" +
		 "  " + os.Args[0] + " homogeneous <portals.json>")
		return
	}

	portals, err := ParseJSONFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", os.Args[2], err)
		os.Exit(1)
	}
	if os.Args[1] == "cobweb" {
		result := LargestCobWeb(portals)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		portalList := []Portal{result[1], result[0]}
		for _, portal := range result[2:] {
			portalList = append(portalList, portal, portalList[len(portalList)-2])
		}
		fmt.Printf("\n[%s]\n", polylineFromPortalList(portalList))
	} else if os.Args[1] == "herringbone" {
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
		if len(os.Args) < 5 {
			fmt.Fprintln(os.Stderr, "Too few arguments for three_corners command")
			os.Exit(1)
		}
		portals1, err := ParseJSONFile(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", os.Args[3], err)
			os.Exit(1)
		}
		portals2, err := ParseJSONFile(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse JSON file %s : %v\n", os.Args[4], err)
			os.Exit(1)
		}

		result := LargestThreeCorner(portals, portals1, portals2)
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
		result, depth := DeepestHomogeneous(portals)
		fmt.Printf("Depth: %d\n", depth+1)
		for i, portal := range result {
			fmt.Printf("%d: %s\n", i, portal.Name)
		}
		polylines := []string{polylineFromPortalList([]Portal{result[0], result[1], result[2]})}
		polylines, _ = appendHomogeneousPolylines(result[0], result[1], result[2], depth, polylines, result[3:])
		fmt.Printf("\n[%s]\n", strings.Join(polylines, ","))
	} else {
		fmt.Fprintf(os.Stderr, "Unknown command: \"%s\"\n", os.Args[1])
		os.Exit(1)
	}
}
