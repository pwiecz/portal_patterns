package main

import "encoding/json"
import "errors"
import "fmt"
import "io/ioutil"
import "os"
import "strconv"

import "github.com/golang/geo/s2"

// Portal description
type Portal struct {
	Name   string
	LatLng s2.Point
}

type portalCoordinates struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

type portalInfo struct {
	Name        string            `json:"title"`
	GUID        string            `json:"guid"`
	Coordinates portalCoordinates `json:"coordinates"`
}

// ParseJSONFile parses file to portal list
func ParseJSONFile(filename string) ([]Portal, error) {
	var portals []Portal
	file, err := os.Open(filename)
	if err != nil {
		return portals, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return portals, err
	}

	var portalsInfo []portalInfo
	if err := json.Unmarshal(bytes, &portalsInfo); err != nil {
		return portals, err
	}
	fmt.Printf("Read %d portals\n", len(portalsInfo))
	for _, portal := range portalsInfo {
		latlng := portal.Coordinates
		lat, err := strconv.ParseFloat(latlng.Lat, 64)
		if err != nil {
			return portals, errors.New("Cannot parse latitude: \"" + latlng.Lat + "\"")
		}
		lng, err := strconv.ParseFloat(latlng.Lng, 64)
		if err != nil {
			return portals, errors.New("Cannot parse longitude: \"" + latlng.Lng + "\"")
		}
		point := s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng))
		portals = append(portals, Portal{Name: portal.Name, LatLng: point})
	}
	return portals, nil
}
