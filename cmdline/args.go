package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/portal_patterns/lib"
)

type portalValue struct {
	LatLngString string
	LatLng       s2.LatLng
}

func (p *portalValue) Set(latLngStr string) error {
	parts := strings.Split(latLngStr, ",")
	if len(parts) != 2 {
		return fmt.Errorf("cannot parse \"%s\" as lat,lng", latLngStr)
	}
	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return err
	}
	lng, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return err
	}
	p.LatLng = s2.LatLngFromDegrees(lat, lng)
	p.LatLngString = latLngStr
	return nil
}
func (p portalValue) String() string {
	return p.LatLngString
}

func portalToIndex(arg portalValue, portals []lib.Portal) int {
	if arg.LatLngString == "" {
		return -1
	}
	result := -1
	found := false
	for j, portal := range portals {
		if s2.PointFromLatLng(arg.LatLng).ApproxEqual(s2.PointFromLatLng(portal.LatLng)) {
			if found {
				log.Fatalf("found more than one portal matching the specified corner portal: %s", arg.LatLngString)
			}
			result = j
			found = true
		}
	}
	if !found {
		log.Fatalf("cound not find portal %s on the provided list of portals", arg.LatLngString)
	}
	return result
}

type portalsValue []portalValue

func (p *portalsValue) Set(latLngStr string) error {
	var portal portalValue
	if err := portal.Set(latLngStr); err != nil {
		return err
	}
	*p = append(*p, portal)
	return nil
}

func (p portalsValue) String() string {
	var portalStrings []string
	for _, portal := range p {
		portalStrings = append(portalStrings, portal.String())
	}
	return strings.Join(portalStrings, ";")
}

func portalsToIndices(arg portalsValue, portals []lib.Portal) []int {
	var indices []int
	for _, portal := range arg {
		indices = append(indices, portalToIndex(portal, portals))
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
		return fmt.Errorf("cannot parse \"%s\" as a 16bit unsigned int", numStr)
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
