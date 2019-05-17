package main

type bestTriangleHeightScorer struct {
	scores [][][]float32
}

type bestTriangleHeightTriangleScorer struct {
	scores     [][][]float32
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
}

type bestTriangleHeightTopLevelScorer struct {
	scores [][][]float32
}

func newBestTriangleHeightScorer(portals []portalData) *bestTriangleHeightScorer {
	scores := make([][][]float32, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		scores = append(scores, make([][]float32, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			scores[i] = append(scores[i], make([]float32, len(portals)))
		}
	}
	return &bestTriangleHeightScorer{
		scores: scores,
	}
}

func (s *bestTriangleHeightScorer) newTriangleScorer(a, b, c portalData) homogeneousTriangleScorer {
	return &bestTriangleHeightTriangleScorer{
		scores:     s.scores,
		a:          a,
		b:          b,
		c:          c,
		abDistance: newDistanceQuery(a.LatLng, b.LatLng),
		acDistance: newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance: newDistanceQuery(b.LatLng, c.LatLng),
	}
}

func (s *bestTriangleHeightScorer) newTopLevelScorer() homogeneousTopLevelScorer {
	return &bestTriangleHeightTopLevelScorer{
		scores: s.scores,
	}
}

func (s *bestTriangleHeightScorer) setTriangleScore(a, b, c portalData, score float32) {
	s.scores[a.Index][b.Index][c.Index] = score
	s.scores[a.Index][c.Index][b.Index] = score
	s.scores[b.Index][a.Index][c.Index] = score
	s.scores[b.Index][c.Index][a.Index] = score
	s.scores[c.Index][a.Index][b.Index] = score
	s.scores[c.Index][b.Index][a.Index] = score
}

func (s *bestTriangleHeightScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.scores[a.Index][b.Index][c.Index]
}

func (s *bestTriangleHeightTriangleScorer) scoreFirstLevelTriangle(p portalData) float32 {
	return float32(
		float64Min(
			s.abDistance.Distance(p.LatLng).Radians(),
			float64Min(
				s.acDistance.Distance(p.LatLng).Radians(),
				s.bcDistance.Distance(p.LatLng).Radians())) * radiansToMeters)
}

func (s *bestTriangleHeightTriangleScorer) scoreHighLevelTriangle(p portalData) float32 {
	return float32Min(
		s.scores[p.Index][s.a.Index][s.b.Index],
		float32Min(
			s.scores[p.Index][s.a.Index][s.c.Index],
			s.scores[p.Index][s.b.Index][s.c.Index]))
}

func (s *bestTriangleHeightTopLevelScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.scores[a.Index][b.Index][c.Index]
}
