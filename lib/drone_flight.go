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
	numKeysNeeded int
	numJumps      int
	queueIndex    int
	index         portalIndex
	prev          portalIndex
}
type droneFlightPrioQueue struct {
	items           []*droneFlightPrioQueueItem
	optimizeNumKeys bool
}

func (pq droneFlightPrioQueue) Len() int { return len(pq.items) }
func (pq droneFlightPrioQueue) Less(i, j int) bool {
	if pq.optimizeNumKeys && pq.items[i].numKeysNeeded != pq.items[j].numKeysNeeded {
		return pq.items[i].numKeysNeeded < pq.items[j].numKeysNeeded
	}
	return pq.items[i].numJumps < pq.items[j].numJumps
}
func (pq droneFlightPrioQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].queueIndex = i
	pq.items[j].queueIndex = j
}
func (pq *droneFlightPrioQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*droneFlightPrioQueueItem)
	item.queueIndex = n
	pq.items = append(pq.items, item)
}
func (pq *droneFlightPrioQueue) Pop() interface{} {
	n := len(pq.items)
	top := pq.items[n-1]
	pq.items[n-1] = nil
	pq.items = pq.items[:n-1]
	return top
}

type droneFlightNeighbour struct {
	index     portalIndex
	keyNeeded bool
}

type longestDroneFlightQuery struct {
	neighbours     [][]droneFlightNeighbour
	portalDistance func(portalIndex, portalIndex) float64
	queue          fifo
	visited        []bool
}

func newLongestDroneFlightQuery(neighbours [][]droneFlightNeighbour, portalDistance func(portalIndex, portalIndex) float64) *longestDroneFlightQuery {
	q := &longestDroneFlightQuery{
		neighbours:     neighbours,
		portalDistance: portalDistance,
		visited:        make([]bool, len(neighbours)),
	}
	return q
}

// Returns most distant target portal reachable from the start portal.
// If end is != invalidPortalIndex returns either the end or invalidPortalIndex depending whether
// there is path from start to end.
func (q *longestDroneFlightQuery) longestFlightFrom(start, end portalIndex) (portalIndex, float64) {
	bestDistance := 0.0
	bestEndPortal := start
	for i := 0; i < len(q.neighbours); i++ {
		q.visited[i] = false
	}
	q.queue.Reset()
	q.queue.Enqueue(start)
	q.visited[start] = true
mainloop:
	for !q.queue.Empty() {
		p := q.queue.Dequeue()
		for _, n := range q.neighbours[p] {
			if q.visited[n.index] {
				continue
			}
			q.queue.Enqueue(n.index)
			q.visited[n.index] = true
			distance := q.portalDistance(n.index, start)
			if n.index == end {
				bestEndPortal = end
				bestDistance = distance
				break mainloop
			}
			if distance > bestDistance {
				bestEndPortal = n.index
				bestDistance = distance
			}
		}
	}
	if end != invalidPortalIndex && bestEndPortal != end {
		return invalidPortalIndex, 0
	}
	return bestEndPortal, bestDistance
}

func (q *longestDroneFlightQuery) optimalFlight(start, end portalIndex, optimizeNumKeys bool) ([]portalIndex, []portalIndex) {
	if start == invalidPortalIndex || end == invalidPortalIndex {
		panic(fmt.Errorf("%d, %d", int(start), int(end)))
	}
	queueItems := make([]droneFlightPrioQueueItem, len(q.neighbours))
	queue := droneFlightPrioQueue{
		items:           make([]*droneFlightPrioQueueItem, 0, len(q.neighbours)),
		optimizeNumKeys: optimizeNumKeys}
	for i := 0; i < len(q.neighbours); i++ {
		q.visited[i] = false
		queueItems[i].index = portalIndex(i)
		queueItems[i].prev = invalidPortalIndex
		queueItems[i].numKeysNeeded = len(q.neighbours) + 1
		queueItems[i].numJumps = len(q.neighbours) + 1
		if portalIndex(i) == start {
			queueItems[i].numKeysNeeded = 0
			queueItems[i].numJumps = 0
		}
		queueItems[i].queueIndex = i
		queue.items = append(queue.items, &queueItems[i])
	}
	heap.Init(&queue)
	for queue.Len() > 0 {
		p := heap.Pop(&queue).(*droneFlightPrioQueueItem)
		if q.visited[p.index] || p.numKeysNeeded > len(q.neighbours) {
			continue
		}
		q.visited[p.index] = true
		if p.index == end {
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
			betterPath := false
			if optimizeNumKeys {
				if keysNeeded < queueItems[n.index].numKeysNeeded ||
					(keysNeeded == queueItems[n.index].numKeysNeeded && numJumps < queueItems[n.index].numJumps) {
					betterPath = true
				}
			} else {
				if numJumps < queueItems[n.index].numJumps {
					betterPath = true
				}
			}
			if betterPath {
				queueItems[n.index].numKeysNeeded = keysNeeded
				queueItems[n.index].numJumps = numJumps
				queueItems[n.index].prev = p.index
				heap.Fix(&queue, queueItems[n.index].queueIndex)
			}
		}
	}
	bestPath := []portalIndex{end}
	keysNeeded := []portalIndex{}
	for {
		lastPortal := bestPath[len(bestPath)-1]
		if prev := queueItems[lastPortal].prev; prev != invalidPortalIndex {
			bestPath = append(bestPath, prev)
			if queueItems[lastPortal].numKeysNeeded > queueItems[prev].numKeysNeeded {
				keysNeeded = append(keysNeeded, lastPortal)
			}
		} else {
			break
		}
	}
	return bestPath, keysNeeded
}

const droneFlightNeighbourCellRange = 500
const droneFlightMaxRange = 1250

func prepareDroneGraph(portalsData []portalData, useLongJumps bool, reverseRoute bool) [][]droneFlightNeighbour {
	cellPortals := make(map[s2.CellID][]portalData)
	portalCells := make([]s2.CellID, len(portalsData))
	for _, p := range portalsData {
		cellId := s2.CellFromPoint(p.LatLng).ID()
		if cellId.Level() < 16 {
			panic(fmt.Errorf("got cell level: %d", cellId.Level()))
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
	if params.startPortalIndex == invalidPortalIndex && params.endPortalIndex != invalidPortalIndex {
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

	// First find most distant pair of portals reachable from one another.
	// We assume that there is not better solution for a different pair with exactly
	// the same distance.
	bestDistance := -1.0
	bestStart, bestEnd := invalidPortalIndex, invalidPortalIndex
	for _, p := range portalsData {
		if params.startPortalIndex != invalidPortalIndex && p.Index != params.startPortalIndex {
			continue
		}
		end, distance := q.longestFlightFrom(p.Index, params.endPortalIndex)
		if end != invalidPortalIndex && distance > bestDistance {
			bestDistance = distance
			bestStart, bestEnd = p.Index, end
		}
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			params.progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	if bestStart == invalidPortalIndex || bestEnd == invalidPortalIndex {
		return nil, nil
	}
	// Now find the most optimal path between those two portals (in both ways).
	// It's way faster two split the search into two parts, as the first one
	// can be calculated using a trivial DFS on the reachability graph using
	// only a simple FIFO queue instead of priority queue.
	bestPath, bestKeysNeeded := q.optimalFlight(bestStart, bestEnd, params.optimizeNumKeys)
	if params.startPortalIndex == invalidPortalIndex {
		path, keysNeeded := q.optimalFlight(bestEnd, bestStart, params.optimizeNumKeys)
		if len(path) > 1 {
			if params.optimizeNumKeys {
				if len(keysNeeded) < len(bestKeysNeeded) || (len(keysNeeded) == len(bestKeysNeeded) && len(path) < len(bestPath)) {
					bestPath, bestKeysNeeded = path, keysNeeded
				}
			} else {
				if len(path) < len(bestPath) {
					bestPath, bestKeysNeeded = path, keysNeeded
				}
			}
		}
	}
	params.progressFunc(numIndexEntries, numIndexEntries)
	if len(bestPath) < 2 {
		return nil, nil
	}

	if reverseRoute {
		reverse(bestPath)
	}
	bestPortalPath := []Portal(nil)
	for i := len(bestPath) - 1; i >= 0; i-- {
		bestPortalPath = append(bestPortalPath, portals[bestPath[i]])
	}
	bestPortalKeysNeeded := []Portal(nil)
	for _, index := range bestKeysNeeded {
		bestPortalKeysNeeded = append(bestPortalKeysNeeded, portals[index])
	}
	return bestPortalPath, bestPortalKeysNeeded
}
