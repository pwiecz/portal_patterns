package lib

import "fmt"
import "math"
import "strings"
import "image"
import "image/png"
import "image/color"
import "os"
import "github.com/golang/geo/s2"
import "github.com/golang/geo/r2"

import "github.com/pwiecz/portal_patterns/lib/r2geo"

type portalIndex uint16

const invalidPortalIndex portalIndex = math.MaxUint16

type portalData struct {
	Index  portalIndex
	LatLng r2.Point
}

/*func portalsToPortalData(portals []Portal) []portalData {
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{
			Index: portalIndex(i),
			LatLng: s2.PointFromLatLng(portal.LatLng)})
	}
	return portalsData
}*/

func portalsToPortalData(portals []Portal) []portalData {
	chq := s2.NewConvexHullQuery()
	for _, portal := range portals {
		chq.AddPoint(s2.PointFromLatLng(portal.LatLng))
	}
	hq := chq.ConvexHull()
	centroid := s2.LatLngFromPoint(hq.Centroid())
	sinCLng := math.Sin(centroid.Lng.Radians())
	cosCLng := math.Cos(centroid.Lng.Radians())
	portalsData := make([]portalData, 0, len(portals))
	minX, minY, maxX, maxY := 1000., 1000., -1000., -1000.
	for i, portal := range portals {
		cosC := sinCLng*math.Sin(portal.LatLng.Lng.Radians()) +
			cosCLng*math.Cos(portal.LatLng.Lng.Radians())*math.Cos(centroid.Lat.Radians()-portal.LatLng.Lat.Radians())
		x := math.Cos(portal.LatLng.Lng.Radians()) * math.Sin(centroid.Lat.Radians()-portal.LatLng.Lat.Radians()) / cosC
		y := (cosCLng*math.Sin(portal.LatLng.Lng.Radians()) - sinCLng*math.Cos(portal.LatLng.Lng.Radians())*math.Cos(centroid.Lat.Radians()-portal.LatLng.Lat.Radians())) / cosC
		minX, minY = math.Min(x, minX), math.Min(y, minY)
		maxX, maxY = math.Max(x, maxX), math.Max(y, maxY)
		fmt.Printf("x:%f, y:%f\n", x, y)
		portalsData = append(portalsData, portalData{
			Index:  portalIndex(i),
			LatLng: r2.Point{X: x, Y: y},
		})
	}
	img := image.NewGray(image.Rect(0, 0, 1000, 1000))
	for i, portal := range portalsData {
		x := (portal.LatLng.X - minX) / (maxX - minX)
		y := (portal.LatLng.Y - minY) / (maxY - minY)
		portalsData[i].LatLng.X = x
		portalsData[i].LatLng.Y = y
		img.Set((int)(x*1000), (int)(y*1000), color.Gray{255})
	}
	f, err := os.Create("/home/piotr/portals.png")
	if err != nil {
		panic(err)
	}
	png.Encode(f, img)
	return portalsData
}

const invalidLength uint16 = math.MaxUint16

type bestSolution struct {
	Index  portalIndex
	Length uint16
}

func portalsInsideWedge(portals []portalData, a, b, c portalData, result []portalData) []portalData {
	wedge := r2geo.NewTriangleWedgeQuery(a.LatLng, b.LatLng, c.LatLng)
	result = result[:0]
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			wedge.ContainsPoint(p.LatLng) {
			result = append(result, p)
		}
	}
	return result
}

// returns a subset of portals from portals that lie inside wedge ab, ac.
// It reorders the input portals slice and returns its subslice
func partitionPortalsInsideWedge(portals []portalData, a, b, c portalData) []portalData {
	wedge := r2geo.NewTriangleWedgeQuery(a.LatLng, b.LatLng, c.LatLng)
	length := len(portals)
	for i := 0; i < length; {
		p := portals[i]
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			wedge.ContainsPoint(p.LatLng) {
			i++
		} else {
			portals[i], portals[length-1] = portals[length-1], portals[i]
			length--
		}
	}
	return portals[:length]
}

func portalsInsideTriangle(portals []portalData, a, b, c portalData, result []portalData) []portalData {
	triangle := r2geo.NewTriangleQuery(a.LatLng, b.LatLng, c.LatLng)
	result = result[:0]
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			triangle.ContainsPoint(p.LatLng) {
			result = append(result, p)
		}
	}
	return result
}

func float64Min(v0, v1 float64) float64 {
	if v0 < v1 {
		return v0
	}
	return v1
}
func float64Max(v0, v1 float64) float64 {
	if v0 > v1 {
		return v0
	}
	return v1
}
func float32Min(v0, v1 float32) float32 {
	if v0 < v1 {
		return v0
	}
	return v1
}

func hasAllIndicesInTheTriple(indices []int, a, b, c int) bool {
	for _, index := range indices {
		if a != index && b != index && c != index {
			return false
		}
	}
	return true
}

func hasAllIndicesInThePair(indices []int, a, b int) bool {
	for _, index := range indices {
		if a != index && b != index {
			return false
		}
	}
	return true
}

func pointToJSONCoords(point s2.Point) string {
	return latLngToJSONCoords(s2.LatLngFromPoint(point))
}
func latLngToJSONCoords(latLng s2.LatLng) string {
	return fmt.Sprintf(`{"lat":%f,"lng":%f}`, latLng.Lat.Degrees(), latLng.Lng.Degrees())
}
func PolylineFromPortalList(portals []Portal) string {
	var json strings.Builder
	json.WriteString(`{"type":"polyline","latLngs":[`)
	if len(portals) > 0 {
		fmt.Fprint(&json, latLngToJSONCoords(portals[0].LatLng))
		for _, portal := range portals[1:] {
			fmt.Fprintf(&json, ",%s", latLngToJSONCoords(portal.LatLng))
		}
	}
	json.WriteString(`],"color":"#a24ac3"}`)
	return json.String()
}

func PrintProgressBar(done int, total int) {
	const maxWidth = 50
	doneWidth := done * maxWidth / total
	var b strings.Builder
	b.WriteString("\r[")
	for i := 1; i < doneWidth; i++ {
		b.WriteRune('=')
	}
	if done < total {
		b.WriteRune('>')
	} else {
		b.WriteRune('=')
	}
	for i := doneWidth; i < maxWidth; i++ {
		b.WriteRune(' ')
	}
	percent := 100. * float32(done) / float32(total)
	b.WriteString(fmt.Sprintf("] %3.1f%% (%d/%d)", percent, done, total))
	fmt.Print(b.String())
}
