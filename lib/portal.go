package lib

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
)

// Portal - portal with geographic coordinates in s2.Point format
type Portal struct {
	Guid   string
	Name   string
	LatLng s2.LatLng
}

// IndexedPortal - Portal plus a number
type IndexedPortal struct {
	Portal Portal
	Index  int
}

// PortalCoordinates - portal coordinates in textual format
type PortalCoordinates struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

// PortalInfo - portal with geographic coordinated in textual format
type PortalInfo struct {
	Guid        string            `json:"guid"`
	Name        string            `json:"title"`
	Coordinates PortalCoordinates `json:"coordinates"`
}

// ParseFile parses file to portal list.
//
// It tries to guess the file format based on extensions of the file.
func ParseFile(filename string) ([]Portal, error) {
	portalInfo, err := parseFileAsPortalInfo(filename)
	if err != nil {
		return nil, err
	}
	allGuids := make(map[string]struct{})
	for _, p := range portalInfo {
		if _, ok := allGuids[p.Guid]; ok {
			return nil, fmt.Errorf("duplicate guid: \"%s\"", p.Guid)
		}
		allGuids[p.Guid] = struct{}{}
	}
	return portalInfoToPortal(portalInfo)
}

// parseFileAsPortalInfo parses file to list of PortalInfo structs
//
// It tries to guess the file format based on extensions of the file.
func parseFileAsPortalInfo(filename string) ([]PortalInfo, error) {
	switch filepath.Ext(filename) {
	case ".csv":
		return parseCSVFileAsPortalInfo(filename)
	case ".json":
		return parseJSONFileAsPortalInfo(filename)
	default:
		return nil, fmt.Errorf("unknown extension of file %s", filename)
	}
}

func fixCSVQuoteEscaping(csvBytes []byte) []byte {
	escapedCsv := make([]byte, 0, len(csvBytes))
	inQuotes := false
	escaping := false
	for _, b := range csvBytes {
		switch b {
		case '"':
			if escaping {
				escapedCsv = append(escapedCsv, '"', '"')
				escaping = false
			} else {
				escapedCsv = append(escapedCsv, '"')
				inQuotes = !inQuotes
			}
		case '\\':
			if escaping {
				escapedCsv = append(escapedCsv, '\\', '\\')
				escaping = false
			} else if inQuotes {
				escaping = true
			} else {
				escapedCsv = append(escapedCsv, '\\')
			}
		default:
			if escaping {
				escapedCsv = append(escapedCsv, '\\', b)
			} else {
				escapedCsv = append(escapedCsv, b)
			}
		}
	}
	return escapedCsv
}

func parseCSVFileAsPortalInfo(filename string) ([]PortalInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// Fix quote escaping from \" to ""
	bytes = fixCSVQuoteEscaping(bytes)
	fileStr := string(bytes)

	r := csv.NewReader(strings.NewReader(fileStr))
	var portals []PortalInfo
	lineNo := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error: %v, in line %d", err, lineNo+1)
		}
		if len(record) != 4 {
			return nil, fmt.Errorf("unexpected number of fields: %d in line %d", len(record), lineNo+1)
		}
		_, err = strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, errors.New("cannot parse latitude: \"" + record[2] + "\"")
		}
		_, err = strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, errors.New("cannot parse longitude: \"" + record[3] + "\"")
		}
		portalCoordinates := PortalCoordinates{Lat: record[2], Lng: record[3]}
		portals = append(portals, PortalInfo{Guid: record[0], Name: record[1], Coordinates: portalCoordinates})
		if len(portals) >= math.MaxUint16-1 {
			return nil, errors.New("too many portals")
		}

		lineNo++
	}
	return portals, nil
}

func parseJSONFileAsPortalInfo(filename string) ([]PortalInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var portals []PortalInfo
	if err := json.Unmarshal(bytes, &portals); err != nil {
		return nil, err
	}
	return portals, nil
}

func portalInfoToPortal(portalInfo []PortalInfo) ([]Portal, error) {
	portals := make([]Portal, 0, len(portalInfo))
	for _, portal := range portalInfo {
		latlng := portal.Coordinates
		lat, err := strconv.ParseFloat(latlng.Lat, 64)
		if err != nil {
			return nil, errors.New("cannot parse latitude: \"" + latlng.Lat + "\"")
		}
		lng, err := strconv.ParseFloat(latlng.Lng, 64)
		if err != nil {
			return nil, errors.New("cannot parse longitude: \"" + latlng.Lng + "\"")
		}
		point := s2.LatLngFromDegrees(lat, lng)
		portals = append(portals, Portal{Guid: portal.Guid, Name: portal.Name, LatLng: point})
		if len(portals) >= math.MaxUint16-1 {
			return nil, errors.New("too many portals")
		}
	}
	return portals, nil
}
