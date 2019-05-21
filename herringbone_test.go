package main

import "testing"

func isCorrectHerringbone(b0, b1 Portal, backbone []Portal) bool {
	if len(backbone) <= 1 {
		return true
	}
	triangle := newTriangleQuery(b0.LatLng, b1.LatLng, backbone[0].LatLng)
	if !triangle.ContainsPoint(backbone[1].LatLng) {
		return false
	}
	return isCorrectHerringbone(b0, b1, backbone[1:])
}

func checkValidHerringboneResult(expectedLength int, b0, b1 Portal, backbone []Portal, t *testing.T) {
	if len(backbone) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(backbone))
	}
	if !isCorrectHerringbone(b0, b1, backbone) {
		t.Errorf("Result is not correct herringbone fielding")
	}
}

func TestHerringbone(t *testing.T) {
	portals, err := ParseJSONFile("portals_test_herringbone.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 3 {
		t.FailNow()
	}
	b0, b1, backbone := LargestHerringbone(portals)
	checkValidHerringboneResult(32, b0, b1, backbone, t)
}
