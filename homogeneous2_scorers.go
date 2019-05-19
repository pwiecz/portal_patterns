package main

type avoidThinTriangles2Scorer struct {
	minHeight [][][]float32
}
type avoidThinTriangles2TriangleScorer struct {
	minHeight  [][][]float32
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
}

func newAvoidThinTriangles2Scorer(portals []portalData) *avoidThinTriangles2Scorer {
	minHeight := make([][][]float32, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		minHeight = append(minHeight, make([][]float32, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			minHeight[i] = append(minHeight[i], make([]float32, len(portals)))
		}
	}
	return &avoidThinTriangles2Scorer{
		minHeight: minHeight,
	}
}

func (s *avoidThinTriangles2Scorer) newTriangleScorer(a, b, c portalData) homogeneous2TriangleScorer {
	a, b, c = sorted(a, b, c)
	return &avoidThinTriangles2TriangleScorer{
		minHeight:  s.minHeight,
		a:          a,
		b:          b,
		c:          c,
		abDistance: newDistanceQuery(a.LatLng, b.LatLng),
		acDistance: newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance: newDistanceQuery(b.LatLng, c.LatLng),
	}
}
func (s *avoidThinTriangles2Scorer) setTriangleScore(a, b, c uint16, score [6]float32) {
	s.minHeight[a][b][c] = score[0]
	s.minHeight[a][c][b] = score[1]
	s.minHeight[b][a][c] = score[2]
	s.minHeight[b][c][a] = score[3]
	s.minHeight[c][a][b] = score[4]
	s.minHeight[c][b][a] = score[5]
}
func (s *avoidThinTriangles2Scorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[a.Index][b.Index][c.Index]
}

// assuming a,b are ordered(sorted), return sorted triple of (p, a, b)
func merge(p, a, b uint16) (uint16, uint16, uint16) {
	if p < a {
		return p, a, b
	}
	if p < b {
		return a, p, b
	}
	return a, b, p
}
func (s *avoidThinTriangles2TriangleScorer) score(p portalData, level int) float32 {
	if level == 0 {
		// We multiply by radiansToMeters not to obtain any meaningful distance measure
		// (as ChordAngle returns a squared distance anyway), but just to scale the number up
		// to make it fit in float32 precision range.
		return float32(
			float64Min(
				float64(s.abDistance.ChordAngle(p.LatLng)),
				float64Min(
					float64(s.acDistance.ChordAngle(p.LatLng)),
					float64(s.bcDistance.ChordAngle(p.LatLng)))) * radiansToMeters)
	}
	s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
	ss0, ss1, ss2 := indexOrdering(s0, s1, s2, level-1)
	t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
	st0, st1, st2 := indexOrdering(t0, t1, t2, level-1)
	u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
	su0, su1, su2 := indexOrdering(u0, u1, u2, level-1)
	return float32Min(
		s.minHeight[ss0][ss1][ss2],
		float32Min(
			s.minHeight[st0][st1][st2],
			s.minHeight[su0][su1][su2]))
}
