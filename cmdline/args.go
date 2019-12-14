package main

import "fmt"
import "log"
import "strconv"
import "strings"

import "github.com/golang/geo/s2"
import "github.com/pwiecz/portal_patterns/lib"

type portalsValue struct {
	LatLngStrings []string
	Portals       []s2.LatLng
}

func (p *portalsValue) Set(latLngStr string) error {
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
	p.Portals = append(p.Portals, s2.LatLngFromDegrees(lat, lng))
	p.LatLngStrings = append(p.LatLngStrings, latLngStr)
	return nil
}

func (p portalsValue) String() string {
	return strings.Join(p.LatLngStrings, ";")
}

func portalsToIndices(arg portalsValue, portals []lib.Portal) []int {
	var indices []int
	for i, latLng := range arg.Portals {
		found := false
		for j, portal := range portals {
			if s2.PointFromLatLng(latLng).ApproxEqual(s2.PointFromLatLng(portal.LatLng)) {
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

type numberLimitValue struct {
	Value   int
	Exactly bool
}

func (n *numberLimitValue) Set(numLimitStr string) error {
	numStr := strings.TrimPrefix(numLimitStr, "<=")
	num, err := strconv.ParseUint(numStr, 10, 16)
	if err != nil {
		return fmt.Errorf("Cannot parse \"%s\" as a 16bit unsigned int", numStr)
	}
	n.Value = int(num)
	n.Exactly = len(numStr) == len(numLimitStr)
	return nil
}

func (n numberLimitValue) String() string {
	if n.Exactly {
		return strconv.FormatUint(uint64(n.Value), 10)
	}
	return "<=" + strconv.FormatUint(uint64(n.Value), 10)
}
