package lib

import (
	"fmt"
	"sync"
)

type droneFlightResponse struct {
	start, end portalIndex
	distance   float64
}

func longestDroneFlightWorker(
	neighbours [][]droneFlightNeighbour,
	portalDistance func(portalIndex, portalIndex) float64,
	targetPortal portalIndex,
	requestChannel chan portalIndex, responseChannel chan droneFlightResponse,
	wg *sync.WaitGroup) {
	q := newLongestDroneFlightQuery(neighbours, portalDistance)
	for start := range requestChannel {
		end, distance := q.longestFlightFrom(start, targetPortal)
		responseChannel <- droneFlightResponse{
			start:    start,
			end:      end,
			distance: distance,
		}
	}
	wg.Done()
}

func longestDroneFlightMT(portals []Portal, params droneFlightParams) ([]Portal, []Portal) {
	if params.numWorkers < 1 {
		panic(fmt.Errorf("Too few workers: %d", params.numWorkers))
	}
	if len(portals) < 2 {
		panic(fmt.Errorf("Too short portal list: %d", len(portals)))
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

	requestChannel := make(chan portalIndex, params.numWorkers)
	responseChannel := make(chan droneFlightResponse, params.numWorkers)

	var wg sync.WaitGroup
	wg.Add(params.numWorkers)

	for i := 0; i < params.numWorkers; i++ {
		go longestDroneFlightWorker(neighbours, portalDistanceInRadians, params.endPortalIndex,
			requestChannel, responseChannel, &wg)
	}
	go func() {
		for _, p := range portalsData {
			if params.startPortalIndex != invalidPortalIndex && p.Index != params.startPortalIndex {
				continue
			}
			requestChannel <- p.Index
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	numIndexEntries := len(portals)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	params.progressFunc(0, numIndexEntries)

	bestDistance := -1.0
	bestStart, bestEnd := invalidPortalIndex, invalidPortalIndex
	for resp := range responseChannel {
		if resp.distance > bestDistance {
			bestDistance = resp.distance
			bestStart, bestEnd = resp.start, resp.end
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
	q := newLongestDroneFlightQuery(neighbours, portalDistanceInRadians)
	bestPath, bestKeysNeeded := q.optimalFlight(bestStart, bestEnd, params.optimizeNumKeys)
	if params.startPortalIndex == invalidPortalIndex {
		path, keysNeeded := q.optimalFlight(bestEnd, bestStart, params.optimizeNumKeys)
		if path != nil {
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
