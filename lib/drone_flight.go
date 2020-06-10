package lib

import "fmt"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

type fifo struct {
	first, second []portalIndex
}

func (f *fifo) Reset() {
	f.first = f.first[:0]
	f.second = f.second[:0]
}
func (f *fifo) Empty() bool {
	return len(f.first)+len(f.second) == 0
}
func (f *fifo) Enqueue(p portalIndex) {
	f.second = append(f.second, p)
}
func (f *fifo) Dequeue() portalIndex {
	if f.Empty() {
		panic("Empty queue")
	}
	if len(f.first) == 0 {
		for i := len(f.second) - 1; i >= 0; i-- {
			f.first = append(f.first, f.second[i])
		}
		f.second = f.second[:0]
	}
	r := f.first[len(f.first)-1]
	f.first = f.first[:len(f.first)-1]
	return r
}

type longestDroneFlightQuery struct {
	neighbours     [][]portalIndex
	portalDistance func(portalIndex, portalIndex) s1.Angle
	queue          fifo
	prevs          []portalIndex
	visited        []bool
}

func newLongestDroneFlightQuery(neighbours [][]portalIndex, portalDistance func(portalIndex, portalIndex) s1.Angle) *longestDroneFlightQuery {
	return &longestDroneFlightQuery{
		neighbours:     neighbours,
		portalDistance: portalDistance,
		prevs:          make([]portalIndex, len(neighbours)),
		visited:        make([]bool, len(neighbours)),
	}
}

// If end is != invalidPortalIndex return only from from start to end if it exists.
func (q *longestDroneFlightQuery) longestFlightFrom(start, end portalIndex) ([]portalIndex, s1.Angle) {
	bestDistance := s1.Angle(0.)
	bestEndPortal := start
	for i := 0; i < len(q.neighbours); i++ {
		q.prevs[i] = invalidPortalIndex
		q.visited[i] = false
	}
	q.queue.Reset()
	q.queue.Enqueue(start)
	q.visited[start] = true
	for !q.queue.Empty() {
		p := q.queue.Dequeue()
		for _, n := range q.neighbours[p] {
			if q.visited[n] {
				continue
			}
			q.queue.Enqueue(n)
			q.visited[n] = true
			q.prevs[n] = p
			distance := q.portalDistance(n, start)
			if n == end {
				bestEndPortal = n
				bestDistance = distance
				q.queue.Reset()
				break
			}
			if distance > bestDistance {
				bestEndPortal = n
				bestDistance = distance
			}
		}
	}
	if end != invalidPortalIndex && bestEndPortal != end {
		return nil, 0
	}
	bestPath := []portalIndex{bestEndPortal}
	for {
		if prev := q.prevs[bestPath[len(bestPath)-1]]; prev != invalidPortalIndex {
			bestPath = append(bestPath, prev)
		} else {
			break
		}
	}
	return bestPath, bestDistance
}

func LongestDroneFlight(portals []Portal, startIndex, endIndex int, progressFunc func(int, int)) ([]Portal, float64) {
	if len(portals) < 2 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)
	cellPortals := make(map[s2.CellID][]portalData)
	portalCells := make([]s2.CellID, len(portals))
	for _, p := range portalsData {
		cellId := s2.CellFromPoint(p.LatLng).ID()
		if cellId.Level() < 16 {
			panic(fmt.Errorf("Got cell level: %d", cellId.Level()))
		}
		cellId = cellId.Parent(16)
		cellPortals[cellId] = append(cellPortals[cellId], p)
		portalCells[p.Index] = cellId
	}
	neighbours := make([][]portalIndex, len(portals))
	for _, p := range portalsData {
		circle500m := s2.CapFromCenterAngle(p.LatLng, s1.Angle(500/radiansToMeters))
		cellsInCircle := s2.FloodFillRegionCovering(circle500m, portalCells[p.Index])
		hasTheCell := false
		for _, cellId := range cellsInCircle {
			if cellId == portalCells[p.Index] {
				hasTheCell = true
			}
			for _, np := range cellPortals[cellId] {
				if np.Index != p.Index {
					neighbours[p.Index] = append(neighbours[p.Index], np.Index)
				}
			}
		}
		if !hasTheCell {
			panic("no origin cell")
		}
	}

	portalDistanceInRadians := func(i, j portalIndex) s1.Angle {
		return portalsData[i].LatLng.Distance(portalsData[j].LatLng)
	}
	numIndexEntries := len(portals)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	onFilledIndexEntry := func() {
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	progressFunc(0, numIndexEntries)

	q := newLongestDroneFlightQuery(neighbours, portalDistanceInRadians)

	targetPortal := invalidPortalIndex
	if endIndex >= 0 {
		targetPortal = portalIndex(endIndex)
	}

	bestDistance := s1.Angle(0.)
	bestPath := []portalIndex{}
	for _, p := range portalsData {
		if startIndex >= 0 && p.Index != portalIndex(startIndex) {
			continue
		}
		path, distance := q.longestFlightFrom(p.Index, targetPortal)
		if distance > bestDistance || (distance == bestDistance && len(path) < len(bestPath)) {
			bestDistance = distance
			bestPath = path
		}
		onFilledIndexEntry()
	}

	bestPortalPath := []Portal{}
	for i := len(bestPath) - 1; i >= 0; i-- {
		bestPortalPath = append(bestPortalPath, portals[bestPath[i]])
	}
	return bestPortalPath, float64(bestDistance) * radiansToMeters
}
