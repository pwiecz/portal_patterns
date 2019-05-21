package main

type thickTrianglesScorer struct {
	minHeight [][][]float32
}

func newThickTrianglesScorer(numPortals int) *thickTrianglesScorer {
	minHeight := make([][][]float32, 0, numPortals)
	for i := 0; i < numPortals; i++ {
		minHeight = append(minHeight, make([][]float32, 0, numPortals))
		for j := 0; j < numPortals; j++ {
			minHeight[i] = append(minHeight[i], make([]float32, numPortals))
		}
	}
	return &thickTrianglesScorer{
		minHeight: minHeight,
	}
}

type thickTrianglesTriangleScorer struct {
	minHeight  [][][]float32
	maxDepth   int
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
	scorePtrs  [6]*float32
	candidates [6]portalIndex
}

func (s *thickTrianglesScorer) newTriangleScorer(a, b, c portalData, maxDepth int) homogeneousTriangleScorer {
	a, b, c = sorted(a, b, c)
	var scorePtrs [6]*float32
	for level := 2; level <= maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		scorePtrs[level-2] = &s.minHeight[i][j][k]
	}
	candidates := [6]portalIndex{
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1}
	return &thickTrianglesTriangleScorer{
		minHeight:  s.minHeight,
		maxDepth:   maxDepth,
		a:          a,
		b:          b,
		c:          c,
		abDistance: newDistanceQuery(a.LatLng, b.LatLng),
		acDistance: newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance: newDistanceQuery(b.LatLng, c.LatLng),
		scorePtrs:  scorePtrs,
		candidates: candidates,
	}
}
func (s *thickTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[a.Index][b.Index][c.Index]
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
				s.minHeight[s0][s1][s2],
				float32Min(
					s.minHeight[t0][t1][t2],
					s.minHeight[u0][u1][u2]))
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
