package lib

import "math"

// a scorer that picks a solution that maximises minimal height of a triangle
// being part of the final solution.
type thickTrianglesScorer struct {
	minHeight  []float32
	numPortals uint
}

type clumpPortalsScorer struct {
	minDistance []float32
	numPortals  uint
}

func newThickTrianglesScorer(numPortals int) *thickTrianglesScorer {
	numPortals64 := uint(numPortals)
	minHeight := make([]float32, numPortals64*numPortals64*numPortals64)
	return &thickTrianglesScorer{
		minHeight:  minHeight,
		numPortals: numPortals64,
	}
}

func newClumpPortalsScorer(numPortals int) *clumpPortalsScorer {
	numPortals64 := uint(numPortals)
	minDistance := make([]float32, numPortals64*numPortals64*numPortals64)
	for i := 0; i < len(minDistance); i++ {
		minDistance[i] = -math.MaxFloat32
	}
	return &clumpPortalsScorer{
		minDistance: minDistance,
		numPortals:  numPortals64,
	}
}

func (s *thickTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[(uint(a.Index)*s.numPortals+uint(b.Index))*s.numPortals+uint(c.Index)]
}

func (s *clumpPortalsScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minDistance[(uint(a.Index)*s.numPortals+uint(b.Index))*s.numPortals+uint(c.Index)]
}

// scorer for picking the best midpoint of triangle a,b,c
type thickTrianglesTriangleScorer struct {
	scorePtrs  [6]*float32
	minHeight  []float32
	acDistance distanceQuery
	bcDistance distanceQuery
	b          portalData
	c          portalData
	abDistance distanceQuery
	a          portalData
	numPortals uint
	maxDepth   int
	candidates [6]portalIndex
}

type clumpPortalsTriangleScorer struct {
	scorePtrs   [6]*float32
	minDistance []float32
	a           portalData
	b           portalData
	c           portalData
	numPortals  uint
	maxDepth    int
	candidates  [6]portalIndex
}

func (s *thickTrianglesScorer) newTriangleScorer(maxDepth int) homogeneousTriangleScorer {
	return &thickTrianglesTriangleScorer{
		minHeight:  s.minHeight,
		numPortals: s.numPortals,
		maxDepth:   maxDepth,
	}
}

func (s *clumpPortalsScorer) newTriangleScorer(maxDepth int) homogeneousTriangleScorer {
	return &clumpPortalsTriangleScorer{
		minDistance: s.minDistance,
		numPortals:  s.numPortals,
		maxDepth:    maxDepth,
	}
}

func (s *thickTrianglesTriangleScorer) reset(a, b, c portalData, numCandidates int) {
	a, b, c = sorted(a, b, c)
	for level := 2; level <= s.maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		s.scorePtrs[level-2] = &s.minHeight[(uint(i)*s.numPortals+uint(j))*s.numPortals+uint(k)]
	}
	for i := 0; i < 6; i++ {
		s.candidates[i] = invalidPortalIndex - 1
	}
	s.a, s.b, s.c = a, b, c
	s.abDistance = newDistanceQuery(a.LatLng, b.LatLng)
	s.acDistance = newDistanceQuery(a.LatLng, c.LatLng)
	s.bcDistance = newDistanceQuery(b.LatLng, c.LatLng)
}

func (s *clumpPortalsTriangleScorer) reset(a, b, c portalData, numCandidates int) {
	a, b, c = sorted(a, b, c)
	for level := 2; level <= s.maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		s.scorePtrs[level-2] = &s.minDistance[(uint(i)*s.numPortals+uint(j))*s.numPortals+uint(k)]
	}
	for i := 0; i < 6; i++ {
		s.candidates[i] = invalidPortalIndex - 1
	}
	s.a, s.b, s.c = a, b, c
}

func (s *thickTrianglesTriangleScorer) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *clumpPortalsTriangleScorer) getDistance(a, b, c portalIndex) float32 {
	return s.minDistance[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}

// assuming a,b are ordered(sorted), return sorted triple of (p, a, b)
func merge(p, a, b portalIndex) (portalIndex, portalIndex, portalIndex) {
	if p < a {
		return p, a, b
	}
	if p < b {
		return a, p, b
	}
	return a, b, p
}

func (s *thickTrianglesTriangleScorer) scoreCandidate(p portalData) {
	// We multiply by RadiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	lvl2Height := float32(
		min(
			float64(s.abDistance.ChordAngle(p.LatLng)),
			min(
				float64(s.acDistance.ChordAngle(p.LatLng)),
				float64(s.bcDistance.ChordAngle(p.LatLng)))) * RadiansToMeters)
	if lvl2Height > *s.scorePtrs[0] {
		*s.scorePtrs[0] = lvl2Height
		s.candidates[0] = p.Index
	}
	s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
	t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
	u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
	for level := 3; level <= s.maxDepth; level++ {
		si0, si1, si2 := indexOrdering(s0, s1, s2, level-1)
		ti0, ti1, ti2 := indexOrdering(t0, t1, t2, level-1)
		ui0, ui1, ui2 := indexOrdering(u0, u1, u2, level-1)
		minHeight := min(
			s.getHeight(si0, si1, si2),
			min(
				s.getHeight(ti0, ti1, ti2),
				s.getHeight(ui0, ui1, ui2)))
		if minHeight == 0 {
			break
		}
		if minHeight > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minHeight
			s.candidates[level-2] = p.Index
		}
	}
}
func (s *clumpPortalsTriangleScorer) scoreCandidate(p portalData) {
	// We multiply by RadiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	minDistance := -float32(
		min(
			distance(s.a, p),
			min(
				distance(s.b, p),
				distance(s.c, p))) * RadiansToMeters)
	if minDistance > *s.scorePtrs[0] {
		*s.scorePtrs[0] = minDistance
		s.candidates[0] = p.Index
	}
	s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
	t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
	u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
	for level := 3; level <= s.maxDepth; level++ {
		si0, si1, si2 := indexOrdering(s0, s1, s2, level-1)
		sDist := s.getDistance(si0, si1, si2)
		if sDist == -math.MaxFloat32 {
			return
		}
		ti0, ti1, ti2 := indexOrdering(t0, t1, t2, level-1)
		tDist := s.getDistance(ti0, ti1, ti2)
		if tDist == -math.MaxFloat32 {
			return
		}
		ui0, ui1, ui2 := indexOrdering(u0, u1, u2, level-1)
		uDist := s.getDistance(ui0, ui1, ui2)
		if uDist == -math.MaxFloat32 {
			return
		}
		dist := minDistance + sDist + tDist + uDist
		if dist > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = dist
			s.candidates[level-2] = p.Index
		}
	}
}

func (s *thickTrianglesTriangleScorer) bestMidpoints() [6]portalIndex {
	return s.candidates
}
func (s *clumpPortalsTriangleScorer) bestMidpoints() [6]portalIndex {
	return s.candidates
}
