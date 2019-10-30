package main

import "fmt"
import "log"
import "strconv"
import "strings"

import "github.com/golang/geo/s2"
import "github.com/pwiecz/portal_patterns/lib"

type PortalsValue struct {
	LatLngStrings []string
	Portals       []s2.Point
}

func (p *PortalsValue) Set(latLngStr string) error {
	parts := strings.Split(latLngStr, ",")
	if len(parts) != 2 {
		return fmt.Errorf("Cannot parse \"%s\" as lat,lng", latLngStr)
	}
	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return err
	}
	lng, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return err
	}
	p.Portals = append(p.Portals, s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng)))
	p.LatLngStrings = append(p.LatLngStrings, latLngStr)
	return nil
}

func (p PortalsValue) String() string {
	return strings.Join(p.LatLngStrings, ";")
}

func portalsToIndices(arg PortalsValue, portals []lib.Portal) []int {
	var indices []int
	for i, latLng := range arg.Portals {
		found := false
		for j, portal := range portals {
			if latLng.ApproxEqual(portal.LatLng) {
				if found {
					log.Fatalf("found more than one portal matching the specified corner portal: %s", arg.LatLngStrings[i])
				}
				indices = append(indices, j)
				found = true
			}
		}
		if !found {
			log.Fatalf("cound not find portal %s on the provided list of portals", arg.LatLngStrings[i])
		}
	}
	return indices

}
