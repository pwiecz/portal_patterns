package lib

type thickTrianglesScorer struct {
	minHeight    []float32
	numPortals   uint
	numPortalsSq uint
}

func newThickTrianglesScorer(numPortals int) *thickTrianglesScorer {
	numPortals64 := uint(numPortals)
	minHeight := make([]float32, numPortals64*numPortals64*numPortals64)
	return &thickTrianglesScorer{
		minHeight:    minHeight,
		numPortals:   numPortals64,
		numPortalsSq: numPortals64 * numPortals64,
	}
}

type thickTrianglesTriangleScorer struct {
	minHeight    []float32
	numPortals   uint
	numPortalsSq uint
	maxDepth     int
	a, b, c      portalData
	abDistance   distanceQuery
	acDistance   distanceQuery
	bcDistance   distanceQuery
	scorePtrs    [6]*float32
	candidates   [6]portalIndex
}

func (s *thickTrianglesScorer) newTriangleScorer(a, b, c portalData, maxDepth int) homogeneousTriangleScorer {
	a, b, c = sorted(a, b, c)
	var scorePtrs [6]*float32
	for level := 2; level <= maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		scorePtrs[level-2] = &s.minHeight[uint(i)*s.numPortalsSq+uint(j)*s.numPortals+uint(k)]
	}
	candidates := [6]portalIndex{
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1}
	return &thickTrianglesTriangleScorer{
		minHeight:    s.minHeight,
		numPortals:   s.numPortals,
		numPortalsSq: s.numPortalsSq,
		maxDepth:     maxDepth,
		a:            a,
		b:            b,
		c:            c,
		abDistance:   newDistanceQuery(a.LatLng, b.LatLng),
		acDistance:   newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance:   newDistanceQuery(b.LatLng, c.LatLng),
		scorePtrs:    scorePtrs,
		candidates:   candidates,
	}
}
func (s *thickTrianglesTriangleScorer) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[uint(a)*s.numPortalsSq+uint(b)*s.numPortals+uint(c)]
}
func (s *thickTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[uint(a.Index)*s.numPortalsSq+uint(b.Index)*s.numPortals+uint(c.Index)]
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
	for level := 2; level <= s.maxDepth; level++ {
		var minHeight float32
		if level == 2 {
			// We multiply by radiansToMeters not to obtain any meaningful distance measure
			// (as ChordAngle returns a squared distance anyway), but just to scale the number up
			// to make it fit in float32 precision range.
			minHeight = float32(
				float64Min(
					float64(s.abDistance.ChordAngle(p.LatLng)),
					float64Min(
						float64(s.acDistance.ChordAngle(p.LatLng)),
						float64(s.bcDistance.ChordAngle(p.LatLng)))) * radiansToMeters)
		} else {
			s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
			s0, s1, s2 = indexOrdering(s0, s1, s2, level-1)
			t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
			t0, t1, t2 = indexOrdering(t0, t1, t2, level-1)
			u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
			u0, u1, u2 = indexOrdering(u0, u1, u2, level-1)
			minHeight = float32Min(
				s.getHeight(s0, s1, s2),
				float32Min(
					s.getHeight(t0, t1, t2),
					s.getHeight(u0, u1, u2)))
		}
		if minHeight == 0 {
			break
		}
		if minHeight > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minHeight
			s.candidates[level-2] = p.Index
		}
	}
}

func (s *thickTrianglesTriangleScorer) bestMidpoints() [6]portalIndex {
	return s.candidates
}
