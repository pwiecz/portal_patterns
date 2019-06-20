package main

import "fmt"
import "math"
import "sort"
import "strconv"

type portalInfoData struct {
	Index int
	Lat   string
	Lng   string
}

// FindColinear - Find portals that have the same latitude or longitude
func FindColinear(portals []PortalInfo) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	numItems := len(portals) * (len(portals) - 1) * (len(portals) - 2) / 2
	everyNth := numItems / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	portalData := make([]portalInfoData, 0, len(portals))
	for i, portal := range portals {
		portalData = append(portalData, portalInfoData{
			Index: i, Lat: portal.Coordinates.Lat, Lng: portal.Coordinates.Lng})
		//		fmt.Println(portal.Name)
	}

	foundColinearPortals := false
	sort.Slice(portalData, func(i, j int) bool { return portalData[i].Lat < portalData[j].Lat })
	{
		numSameLat := 0
		for i, portal := range portalData {
			if portal.Lat == portalData[i-numSameLat].Lat {
				numSameLat++
			} else {
				if numSameLat >= 3 {
					sameLat := portalData[i-numSameLat : i]
					lngs := make([]float64, 0, numSameLat)
					sort.Slice(sameLat, func(i, j int) bool { return sameLat[i].Lng < sameLat[j].Lng })
					fmt.Printf("%d colinear portals:\n", numSameLat)
					minLngSpan := math.MaxFloat64

					for j, colinearPortalData := range sameLat {
						lng, _ := strconv.ParseFloat(colinearPortalData.Lng, 64)
						lngs = append(lngs, lng)
						if j >= 2 {
							minLngSpan = math.Min(minLngSpan, lng-lngs[j-2])
						}
						colinearPortal := portals[colinearPortalData.Index]
						fmt.Printf("%d: %s: %s,%s\n", j+1, colinearPortal.Name, colinearPortalData.Lat, colinearPortalData.Lng)
					}
					fmt.Printf("Longitude span: %f\n\n", minLngSpan)
					foundColinearPortals = true
				}
				numSameLat = 1
			}
		}
	}
	sort.Slice(portalData, func(i, j int) bool { return portalData[i].Lng < portalData[j].Lng })
	{
		numSameLng := 0
		for i, portal := range portalData {
			if portal.Lng == portalData[i-numSameLng].Lng {
				numSameLng++
			} else {
				if numSameLng >= 3 {
					sameLng := portalData[i-numSameLng : i]
					lats := make([]float64, 0, numSameLng)
					sort.Slice(sameLng, func(i, j int) bool { return sameLng[i].Lat < sameLng[j].Lat })
					fmt.Printf("%d colinear portals:\n", numSameLng)
					minLatSpan := math.MaxFloat64
					for j, colinearPortalData := range sameLng {
						lat, _ := strconv.ParseFloat(colinearPortalData.Lat, 64)
						lats = append(lats, lat)
						if j >= 2 {
							minLatSpan = math.Min(minLatSpan, lat-lats[j-1])
						}
						colinearPortal := portals[colinearPortalData.Index]
						fmt.Printf("%d: %s: %s,%s\n", j+1, colinearPortal.Name, colinearPortalData.Lat, colinearPortalData.Lng)
					}
					fmt.Printf("Latitude span: %f\n\n", minLatSpan)
					foundColinearPortals = true
				}
				numSameLng = 1
			}
		}
	}
	if !foundColinearPortals {
		fmt.Println("No colinear portals found")
	}
}
