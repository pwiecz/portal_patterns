package lib

import "math"

// a scorer that picks a solution that maximises minimal height of a triangle
// being part of the final solution.
type thickTrianglesScorer struct {
	minHeight  []float32
	numPortals uint
}

type clampPortalsScorer struct {
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

func newClampPortalsScorer(numPortals int) *clampPortalsScorer {
	numPortals64 := uint(numPortals)
	minDistance := make([]float32, numPortals64*numPortals64*numPortals64)
	for i := 0; i < len(minDistance); i++ {
		minDistance[i] = -math.MaxFloat32
	}
	return &clampPortalsScorer{
		minDistance: minDistance,
		numPortals:  numPortals64,
	}
}

func (s *thickTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[(uint(a.Index)*s.numPortals+uint(b.Index))*s.numPortals+uint(c.Index)]
}

func (s *clampPortalsScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minDistance[(uint(a.Index)*s.numPortals+uint(b.Index))*s.numPortals+uint(c.Index)]
}

// scorer for picking the best midpoint of triangle a,b,c
type thickTrianglesTriangleScorer struct {
	minHeight  []float32
	numPortals uint
	maxDepth   int
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
	scorePtrs  [6]*float32
	candidates [6]portalIndex
}

type clampPortalsTriangleScorer struct {
	minDistance []float32
	numPortals  uint
	maxDepth    int
	a, b, c     portalData
	scorePtrs   [6]*float32
	candidates  [6]portalIndex
}

// scorer for picking the best midpoint of triangle a,b,c for given depth
type thickTrianglesDepthTriangleScorer struct {
	minHeight  []float32
	numPortals uint
	depth      uint16
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
	scorePtr   *float32
	candidate  portalIndex
}
type clampPortalsDepthTriangleScorer struct {
	minDistance []float32
	numPortals  uint
	depth       uint16
	a, b, c     portalData
	scorePtr    *float32
	candidate   portalIndex
}

// scorer for picking the best midpoint of triangle a,b,c for perfect homogeneous fields
type thickTrianglesTriangleScorerPerfect struct {
	minHeight            []float32
	numPortals           uint
	maxDepth             int
	validLevel2Candidate bool
	a, b, c              portalData
	abDistance           distanceQuery
	acDistance           distanceQuery
	bcDistance           distanceQuery
	scorePtrs            [6]*float32
	candidates           [6]portalIndex
}
type clampPortalsTriangleScorerPerfect struct {
	minDistance          []float32
	numPortals           uint
	maxDepth             int
	validLevel2Candidate bool
	a, b, c              portalData
	scorePtrs            [6]*float32
	candidates           [6]portalIndex
}

func (s *thickTrianglesScorer) newTriangleScorer(maxDepth int, perfect bool) homogeneousTriangleScorer {
	if perfect {
		return &thickTrianglesTriangleScorerPerfect{
			minHeight:  s.minHeight,
			numPortals: s.numPortals,
			maxDepth:   maxDepth,
		}
	}
	return &thickTrianglesTriangleScorer{
		minHeight:  s.minHeight,
		numPortals: s.numPortals,
		maxDepth:   maxDepth,
	}
}

func (s *thickTrianglesScorer) newTriangleScorerForDepth(depth uint16) homogeneousDepthTriangleScorer {
	return &thickTrianglesDepthTriangleScorer{
		minHeight:  s.minHeight,
		numPortals: s.numPortals,
		depth:      depth,
	}
}

func (s *clampPortalsScorer) newTriangleScorer(maxDepth int, perfect bool) homogeneousTriangleScorer {
	if perfect {
		return &clampPortalsTriangleScorerPerfect{
			minDistance: s.minDistance,
			numPortals:  s.numPortals,
			maxDepth:    maxDepth,
		}
	}
	return &clampPortalsTriangleScorer{
		minDistance: s.minDistance,
		numPortals:  s.numPortals,
		maxDepth:    maxDepth,
	}
}

func (s *clampPortalsScorer) newTriangleScorerForDepth(depth uint16) homogeneousDepthTriangleScorer {
	return &clampPortalsDepthTriangleScorer{
		minDistance: s.minDistance,
		numPortals:  s.numPortals,
		depth:       depth,
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
func (s *thickTrianglesTriangleScorerPerfect) reset(a, b, c portalData, numCandidates int) {
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
	s.validLevel2Candidate = numCandidates == 1
}

func (s *thickTrianglesDepthTriangleScorer) reset(a, b, c portalData) {
	a, b, c = sorted(a, b, c)
	i, j, k := indexOrdering(a.Index, b.Index, c.Index, int(s.depth))
	s.scorePtr = &s.minHeight[(uint(i)*s.numPortals+uint(j))*s.numPortals+uint(k)]
	s.candidate = invalidPortalIndex - 1
	s.a, s.b, s.c = a, b, c
	if s.depth == 2 {
		s.abDistance = newDistanceQuery(a.LatLng, b.LatLng)
		s.acDistance = newDistanceQuery(a.LatLng, c.LatLng)
		s.bcDistance = newDistanceQuery(b.LatLng, c.LatLng)
	}
}

func (s *clampPortalsTriangleScorer) reset(a, b, c portalData, numCandidates int) {
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
func (s *clampPortalsTriangleScorerPerfect) reset(a, b, c portalData, numCandidates int) {
	a, b, c = sorted(a, b, c)
	for level := 2; level <= s.maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		s.scorePtrs[level-2] = &s.minDistance[(uint(i)*s.numPortals+uint(j))*s.numPortals+uint(k)]
	}
	for i := 0; i < 6; i++ {
		s.candidates[i] = invalidPortalIndex - 1
	}
	s.a, s.b, s.c = a, b, c
	s.validLevel2Candidate = numCandidates == 1
}

func (s *clampPortalsDepthTriangleScorer) reset(a, b, c portalData) {
	a, b, c = sorted(a, b, c)
	i, j, k := indexOrdering(a.Index, b.Index, c.Index, int(s.depth))
	s.scorePtr = &s.minDistance[(uint(i)*s.numPortals+uint(j))*s.numPortals+uint(k)]
	s.candidate = invalidPortalIndex - 1
	s.a, s.b, s.c = a, b, c
}

func (s *thickTrianglesTriangleScorer) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *thickTrianglesTriangleScorerPerfect) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *thickTrianglesDepthTriangleScorer) getHeight(a, b, c portalIndex) float32 {
	return s.minHeight[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *clampPortalsTriangleScorer) getDistance(a, b, c portalIndex) float32 {
	return s.minDistance[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *clampPortalsTriangleScorerPerfect) getDistance(a, b, c portalIndex) float32 {
	return s.minDistance[(uint(a)*s.numPortals+uint(b))*s.numPortals+uint(c)]
}
func (s *clampPortalsDepthTriangleScorer) getDistance(a, b, c portalIndex) float32 {
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

func (s *thickTrianglesDepthTriangleScorer) scoreCandidate(p portalData) {
	if s.depth == 2 {
		// We multiply by radiansToMeters not to obtain any meaningful distance measure
		// (as ChordAngle returns a squared distance anyway), but just to scale the number up
		// to make it fit in float32 precision range.
		lvl2Height := float32(
			float64Min(
				float64(s.abDistance.ChordAngle(p.LatLng)),
				float64Min(
					float64(s.acDistance.ChordAngle(p.LatLng)),
					float64(s.bcDistance.ChordAngle(p.LatLng)))) * radiansToMeters)
		if lvl2Height > *s.scorePtr {
			*s.scorePtr = lvl2Height
			s.candidate = p.Index
		}
	} else {
		s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
		s0, s1, s2 = indexOrdering(s0, s1, s2, int(s.depth-1))
		t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
		t0, t1, t2 = indexOrdering(t0, t1, t2, int(s.depth-1))
		u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
		u0, u1, u2 = indexOrdering(u0, u1, u2, int(s.depth-1))
		minHeight := float32Min(
			s.getHeight(s0, s1, s2),
			float32Min(
				s.getHeight(t0, t1, t2),
				s.getHeight(u0, u1, u2)))
		if minHeight > *s.scorePtr {
			*s.scorePtr = minHeight
			s.candidate = p.Index
		}
	}
}
func (s *clampPortalsDepthTriangleScorer) scoreCandidate(p portalData) {
	// We multiply by radiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	minDistance := -float32(
		float64Min(
			distanceSq(s.a, p),
			float64Min(
				distanceSq(s.b, p),
				distanceSq(s.c, p))) * radiansToMeters)
	if minDistance > *s.scorePtr {
		*s.scorePtr = minDistance
		s.candidate = p.Index
	}
}

func (s *thickTrianglesTriangleScorer) scoreCandidate(p portalData) {
	// We multiply by radiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	lvl2Height := float32(
		float64Min(
			float64(s.abDistance.ChordAngle(p.LatLng)),
			float64Min(
				float64(s.acDistance.ChordAngle(p.LatLng)),
				float64(s.bcDistance.ChordAngle(p.LatLng)))) * radiansToMeters)
	if lvl2Height > *s.scorePtrs[0] {
		*s.scorePtrs[0] = lvl2Height
		s.candidates[0] = p.Index
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
			break
		}
		if minHeight > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minHeight
			s.candidates[level-2] = p.Index
		}
	}
}
func (s *clampPortalsTriangleScorer) scoreCandidate(p portalData) {
	// We multiply by radiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	minDistance := -float32(
		float64Min(
			distanceSq(s.a, p),
			float64Min(
				distanceSq(s.b, p),
				distanceSq(s.c, p))) * radiansToMeters)
	if minDistance > *s.scorePtrs[0] {
		*s.scorePtrs[0] = minDistance
		s.candidates[0] = p.Index
	}
	for level := 3; level <= s.maxDepth; level++ {
		s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
		s0, s1, s2 = indexOrdering(s0, s1, s2, level-1)
		t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
		t0, t1, t2 = indexOrdering(t0, t1, t2, level-1)
		u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
		u0, u1, u2 = indexOrdering(u0, u1, u2, level-1)
		minInsideDistance := float32Min(
			s.getDistance(s0, s1, s2),
			float32Min(
				s.getDistance(t0, t1, t2),
				s.getDistance(u0, u1, u2)))
		if minInsideDistance == -math.MaxFloat32 {
			break
		}
		if minDistance > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minDistance
			s.candidates[level-2] = p.Index
		}
	}
}
func (s *thickTrianglesTriangleScorerPerfect) scoreCandidate(p portalData) {
	if s.validLevel2Candidate {
		// We multiply by radiansToMeters not to obtain any meaningful distance measure
		// (as ChordAngle returns a squared distance anyway), but just to scale the number up
		// to make it fit in float32 precision range.
		lvl2Height := float32(
			float64Min(
				float64(s.abDistance.ChordAngle(p.LatLng)),
				float64Min(
					float64(s.acDistance.ChordAngle(p.LatLng)),
					float64(s.bcDistance.ChordAngle(p.LatLng)))) * radiansToMeters)
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
		if minHeight > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minHeight
			s.candidates[level-2] = p.Index
		}
	}
}
func (s *clampPortalsTriangleScorerPerfect) scoreCandidate(p portalData) {
	// We multiply by radiansToMeters not to obtain any meaningful distance measure
	// (as ChordAngle returns a squared distance anyway), but just to scale the number up
	// to make it fit in float32 precision range.
	minDistance := -float32(
		float64Min(
			distanceSq(s.a, p),
			float64Min(
				distanceSq(s.b, p),
				distanceSq(s.c, p))) * radiansToMeters)
	if s.validLevel2Candidate {
		if minDistance > *s.scorePtrs[0] {
			*s.scorePtrs[0] = minDistance
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
		minInnerDistance := float32Min(
			s.getDistance(s0, s1, s2),
			float32Min(
				s.getDistance(t0, t1, t2),
				s.getDistance(u0, u1, u2)))
		if minInnerDistance == -math.MaxFloat32 {
			continue
		}
		if minDistance > *s.scorePtrs[level-2] {
			*s.scorePtrs[level-2] = minDistance
			s.candidates[level-2] = p.Index
		}
	}
}

func (s *thickTrianglesTriangleScorer) bestMidpoints() [6]portalIndex {
	return s.candidates
}
func (s *clampPortalsTriangleScorer) bestMidpoints() [6]portalIndex {
	return s.candidates
}
func (s *thickTrianglesTriangleScorerPerfect) bestMidpoints() [6]portalIndex {
	return s.candidates
}
func (s *clampPortalsTriangleScorerPerfect) bestMidpoints() [6]portalIndex {
	return s.candidates
}

func (s *thickTrianglesDepthTriangleScorer) bestMidpoint() portalIndex {
	return s.candidate
}
func (s *clampPortalsDepthTriangleScorer) bestMidpoint() portalIndex {
	return s.candidate
}
