package lib

// a scorer that picks a solution that maximises minimal height of a triangle
// being part of the final solution.
type thickTrianglesScorer struct {
	minHeight  []float32
	numPortals uint
}

func newThickTrianglesScorer(numPortals int) *thickTrianglesScorer {
	numPortals64 := uint(numPortals)
	minHeight := make([]float32, numPortals64*numPortals64*numPortals64)
	return &thickTrianglesScorer{
		minHeight:  minHeight,
		numPortals: numPortals64,
	}
}

// scorer for picking the best midpoint of triangle a,b,c
type thickTrianglesTriangleScorer struct {
	minHeight            []float32
	numPortals           uint
	maxDepth             int
	perfect              bool
	validLevel2Candidate bool
	a, b, c              portalData
	abDistance           DistanceQuery
	acDistance           DistanceQuery
	bcDistance           DistanceQuery
	scorePtrs            [6]*float32
	candidates           [6]portalIndex
}

func (s *thickTrianglesScorer) newTriangleScorer(maxDepth int, perfect bool) homogeneousTriangleScorer {
	return &thickTrianglesTriangleScorer{
		minHeight:  s.minHeight,
		numPortals: s.numPortals,
		maxDepth:   maxDepth,
		perfect:    perfect,
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
	s.abDistance = NewDistanceQuery(a.LatLng, b.LatLng)
	s.acDistance = NewDistanceQuery(a.LatLng, c.LatLng)
	s.bcDistance = NewDistanceQuery(b.LatLng, c.LatLng)
	s.validLevel2Candidate = !s.perfect || numCandidates == 1
}
func (s *thickTrianglesTriangleScorer) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *thickTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[(uint(a.Index)*s.numPortals+uint(b.Index))*s.numPortals+uint(c.Index)]
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
	if s.validLevel2Candidate {
		lvl2Height := float32(
			float64Min(
				s.abDistance.DistanceSq(p.LatLng),
				float64Min(
					s.acDistance.DistanceSq(p.LatLng),
					s.bcDistance.DistanceSq(p.LatLng))))
		if lvl2Height > *s.scorePtrs[0] {
			*s.scorePtrs[0] = lvl2Height
			s.candidates[0] = p.Index
		}
	}
	for level := 3; level <= s.maxDepth; level++ {
		s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
		s0, s1, s2 = indexOrdering(s0, s1, s2, level-1)
		t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
		t0, t1, t2 = indexOrdering(t0, t1, t2, level-1)
		u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
		u0, u1, u2 = indexOrdering(u0, u1, u2, level-1)
		minHeight := float32Min(
			s.getHeight(s0, s1, s2),
			float32Min(
				s.getHeight(t0, t1, t2),
				s.getHeight(u0, u1, u2)))
		if minHeight == 0 {
			if s.perfect {
				continue
			} else {
				break
			}
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
