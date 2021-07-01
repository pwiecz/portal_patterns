package main

import (
	"github.com/golang/geo/s2"
	"github.com/pwiecz/portal_patterns/lib"
)

type PortalIndex struct {
	portalShapeIndex     *s2.ShapeIndex
	shapeIndexIdToPortal map[int32]int
}

func NewPortalIndex(portals []lib.Portal) *PortalIndex {
	index := &PortalIndex{
		portalShapeIndex:     s2.NewShapeIndex(),
		shapeIndexIdToPortal: make(map[int32]int),
	}
	for i, portal := range portals {
		portalPoint := s2.PointFromLatLng(portal.LatLng)
		portalCells := s2.SimpleRegionCovering(portalPoint, portalPoint, 30)
		if len(portalCells) != 1 {
			panic(portalCells)
		}
		cell := s2.CellFromCellID(portalCells[0])
		portalID := index.portalShapeIndex.Add(s2.PolygonFromCell(cell))
		index.shapeIndexIdToPortal[portalID] = i
	}
	return index
}

func (i *PortalIndex) ClosestPortal(point s2.Point) (int, bool) {
	opts := s2.NewClosestEdgeQueryOptions().MaxResults(1)
	query := s2.NewClosestEdgeQuery(i.portalShapeIndex, opts)
	target := s2.NewMinDistanceToPointTarget(point)
	result := query.FindEdges(target)
	if len(result) == 0 {
		return 0, false
	}
	shapeID := result[0].ShapeID()
	portalIx, ok := i.shapeIndexIdToPortal[shapeID]
	return portalIx, ok
}
