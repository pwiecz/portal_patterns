package lib

import (
	"container/heap"
	"fmt"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

func LongestDroneFlight(portals []Portal, options ...DroneFlightOption) ([]Portal, []Portal) {
	params := defaultDroneFlightParams()
	for _, option := range options {
		option.apply(&params)
	}
	if params.numWorkers == 1 {
		return longestDroneFlightST(portals, params)
	}
	return longestDroneFlightMT(portals, params)
}

type droneFlightPrioQueueItem struct {
	index         portalIndex
	numKeysNeeded int
	numJumps      int
	prev          portalIndex
	queueIndex    int
}
type droneFlightPrioQueue []*droneFlightPrioQueueItem

func (pq droneFlightPrioQueue) Len() int { return len(pq) }
func (pq droneFlightPrioQueue) Less(i, j int) bool {
	if pq[i].numKeysNeeded != pq[j].numKeysNeeded {
		return pq[i].numKeysNeeded < pq[j].numKeysNeeded
	}
	return pq[i].numJumps < pq[j].numJumps
}
func (pq droneFlightPrioQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].queueIndex = i
	pq[j].queueIndex = j
}
func (pq *droneFlightPrioQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*droneFlightPrioQueueItem)
	item.queueIndex = n
	*pq = append(*pq, item)
}
func (pq *droneFlightPrioQueue) Pop() interface{} {
	n := len(*pq)
	top := (*pq)[n-1]
	(*pq)[n-1] = nil
	*pq = (*pq)[:n-1]
	return top
}

type droneFlightNeighbour struct {
	index     portalIndex
	keyNeeded bool
}

type longestDroneFlightQuery struct {
	neighbours     [][]droneFlightNeighbour
	portalDistance func(portalIndex, portalIndex) float64
	queueItems     []droneFlightPrioQueueItem
	queue          droneFlightPrioQueue
	visited        []bool
}

func newLongestDroneFlightQuery(neighbours [][]droneFlightNeighbour, portalDistance func(portalIndex, portalIndex) float64) *longestDroneFlightQuery {
	q := &longestDroneFlightQuery{
		neighbours:     neighbours,
		portalDistance: portalDistance,
		visited:        make([]bool, len(neighbours)),
		queueItems:     make([]droneFlightPrioQueueItem, len(neighbours)),
	}
	for i := 0; i < len(neighbours); i++ {
		q.queueItems[i].index = portalIndex(i)
	}
	return q
}

// If end is != invalidPortalIndex return only from from start to end if it exists.
func (q *longestDroneFlightQuery) longestFlightFrom(start, end portalIndex) ([]portalIndex, []portalIndex, float64) {
	bestDistance := 0.
	bestNumKeysNeeded := 0
	bestEndPortal := start
	q.queue = q.queue[:0]
	for i := 0; i < len(q.neighbours); i++ {
		q.visited[i] = false
		q.queueItems[i].prev = invalidPortalIndex
		q.queueItems[i].numKeysNeeded = len(q.neighbours) + 1
		if portalIndex(i) == start {
			q.queueItems[i].numKeysNeeded = 0
		}
		q.queueItems[i].queueIndex = i
		q.queue = append(q.queue, &q.queueItems[i])
	}
	heap.Init(&q.queue)
	for q.queue.Len() > 0 {
		p := heap.Pop(&q.queue).(*droneFlightPrioQueueItem)
		if q.visited[p.index] || p.numKeysNeeded > len(q.neighbours) {
			continue
		}
		q.visited[p.index] = true
		if p.index == end {
			bestEndPortal = p.index
			bestDistance = q.portalDistance(p.index, start)
			bestNumKeysNeeded = p.numKeysNeeded
			break
		}
		for _, n := range q.neighbours[p.index] {
			if q.visited[n.index] {
				continue
			}
			keysNeeded := p.numKeysNeeded
			if n.keyNeeded {
				keysNeeded++
			}
			numJumps := p.numJumps + 1
			if keysNeeded < q.queueItems[n.index].numKeysNeeded ||
				(keysNeeded == q.queueItems[n.index].numKeysNeeded && numJumps < q.queueItems[n.index].numJumps) {
				q.queueItems[n.index].numKeysNeeded = keysNeeded
				q.queueItems[n.index].numJumps = numJumps
				q.queueItems[n.index].prev = p.index
				heap.Fix(&q.queue, q.queueItems[n.index].queueIndex)
			}
			distance := q.portalDistance(n.index, start)
			if distance > bestDistance || (distance == bestDistance && p.numKeysNeeded < bestNumKeysNeeded) {
				bestEndPortal = n.index
				bestDistance = distance
				bestNumKeysNeeded = p.numKeysNeeded + keysNeeded
			}
		}
	}
	if end != invalidPortalIndex && bestEndPortal != end {
		return nil, nil, 0
	}
	bestPath := []portalIndex{bestEndPortal}
	keysNeeded := []portalIndex{}
	for {
		lastPortal := bestPath[len(bestPath)-1]
		if prev := q.queueItems[lastPortal].prev; prev != invalidPortalIndex {
			bestPath = append(bestPath, prev)
			if q.queueItems[lastPortal].numKeysNeeded > q.queueItems[prev].numKeysNeeded {
				keysNeeded = append(keysNeeded, lastPortal)
			}
		} else {
			break
		}
	}
	return bestPath, keysNeeded, bestDistance
}

const droneFlightNeighbourCellRange = 500
const droneFlightMaxRange = 1250

func prepareDroneGraph(portalsData []portalData, useLongJumps bool, reverseRoute bool) [][]droneFlightNeighbour {
	cellPortals := make(map[s2.CellID][]portalData)
	portalCells := make([]s2.CellID, len(portalsData))
	for _, p := range portalsData {
		cellId := s2.CellFromPoint(p.LatLng).ID()
		if cellId.Level() < 16 {
			panic(fmt.Errorf("Got cell level: %d", cellId.Level()))
		}
		cellId = cellId.Parent(16)
		cellPortals[cellId] = append(cellPortals[cellId], p)
		portalCells[p.Index] = cellId
	}
	neighbours := make([][]droneFlightNeighbour, len(portalsData))

	for _, p := range portalsData {
		cellsInSmallCircle := make(map[s2.CellID]struct{})
		{
			// A circle for which we don't require key.
			circle := s2.CapFromCenterAngle(p.LatLng, s1.Angle(droneFlightNeighbourCellRange/RadiansToMeters))
			cellsInCircle := s2.FloodFillRegionCovering(circle, portalCells[p.Index])
			for _, cellId := range cellsInCircle {
				cellsInSmallCircle[cellId] = struct{}{}
				for _, np := range cellPortals[cellId] {
					if np.Index == p.Index {
						continue
					}
					if !reverseRoute {
						neighbours[p.Index] = append(neighbours[p.Index], droneFlightNeighbour{index: np.Index})
					} else {
						neighbours[np.Index] = append(neighbours[np.Index], droneFlightNeighbour{index: p.Index})
					}
				}
			}
		}
		if useLongJumps {
			// We need a key to fly to portals in this larger circle.
			circle := s2.CapFromCenterAngle(p.LatLng, s1.Angle(droneFlightMaxRange/RadiansToMeters))
			cellsInCircle := s2.FloodFillRegionCovering(circle, portalCells[p.Index])
			for _, cellId := range cellsInCircle {
				if _, ok := cellsInSmallCircle[cellId]; ok {
					continue
				}
				for _, np := range cellPortals[cellId] {
					if np.Index == p.Index {
						continue
					}
					if p.LatLng.Distance(np.LatLng) > droneFlightMaxRange/RadiansToMeters {
						continue
					}
					if !reverseRoute {
						neighbours[p.Index] = append(neighbours[p.Index], droneFlightNeighbour{index: np.Index, keyNeeded: true})
					} else {
						neighbours[np.Index] = append(neighbours[np.Index], droneFlightNeighbour{index: p.Index, keyNeeded: true})
					}
				}
			}
		}
	}
	return neighbours
}

func longestDroneFlightST(portals []Portal, params droneFlightParams) ([]Portal, []Portal) {
	if len(portals) < 2 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	// If we have specified endIndex and not startIndex it's much faster to find best
	// route from the endIndex using reversed neighbours list, and later reverse the
	// result route.
	reverseRoute := false
	if params.startPortalIndex < 0 && params.endPortalIndex >= 0 {
		reverseRoute = true
		params.startPortalIndex, params.endPortalIndex = params.endPortalIndex, params.startPortalIndex
	}

	neighbours := prepareDroneGraph(portalsData, params.useLongJumps, reverseRoute)
	portalDistanceInRadians := func(i, j portalIndex) float64 {
		return distanceSq(portalsData[i], portalsData[j])
	}
	numIndexEntries := len(portals)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	params.progressFunc(0, numIndexEntries)

	q := newLongestDroneFlightQuery(neighbours, portalDistanceInRadians)

	targetPortal := invalidPortalIndex
	if params.endPortalIndex >= 0 {
		targetPortal = portalIndex(params.endPortalIndex)
	}

	bestDistance := -1.
	bestPath := []portalIndex{}
	bestKeysNeeded := []portalIndex{}
	for _, p := range portalsData {
		if params.startPortalIndex >= 0 && p.Index != portalIndex(params.startPortalIndex) {
			continue
		}
		path, keysNeeded, distance := q.longestFlightFrom(p.Index, targetPortal)
		if distance > bestDistance || (distance == bestDistance && (len(keysNeeded) < len(bestKeysNeeded) || (len(keysNeeded) == len(bestKeysNeeded) && len(path) < len(bestPath)))) {
			bestDistance = distance
			bestKeysNeeded = keysNeeded
			bestPath = path
		}
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			params.progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	params.progressFunc(numIndexEntries, numIndexEntries)

	if reverseRoute {
		reverse(bestPath)
	}
	bestPortalPath := []Portal{}
	for i := len(bestPath) - 1; i >= 0; i-- {
		bestPortalPath = append(bestPortalPath, portals[bestPath[i]])
	}
	bestPortalKeysNeeded := []Portal{}
	for _, index := range bestKeysNeeded {
		bestPortalKeysNeeded = append(bestPortalKeysNeeded, portals[index])
	}
	return bestPortalPath, bestPortalKeysNeeded
}
