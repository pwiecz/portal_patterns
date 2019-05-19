package main

type avoidThinTrianglesScorer struct {
	minHeight [][][]float32
}
type avoidThinTrianglesTriangleScorer struct {
	minHeight  [][][]float32
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
}

type avoidSmallTrianglesScorer struct {
	minArea [][][]float32
}
type avoidSmallTrianglesTriangleScorer struct {
	minArea [][][]float32
	a, b, c portalData
}

func newAvoidThinTrianglesScorer(portals []portalData) *avoidThinTrianglesScorer {
	minHeight := make([][][]float32, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		minHeight = append(minHeight, make([][]float32, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			minHeight[i] = append(minHeight[i], make([]float32, len(portals)))
		}
	}
	return &avoidThinTrianglesScorer{
		minHeight: minHeight,
	}
}
func newAvoidSmallTrianglesScorer(portals []portalData) *avoidSmallTrianglesScorer {
	minArea := make([][][]float32, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		minArea = append(minArea, make([][]float32, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			minArea[i] = append(minArea[i], make([]float32, len(portals)))
		}
	}
	return &avoidSmallTrianglesScorer{
		minArea: minArea,
	}
}

func (s *avoidThinTrianglesScorer) newTriangleScorer(a, b, c portalData) homogeneousTriangleScorer {
	return &avoidThinTrianglesTriangleScorer{
		minHeight:  s.minHeight,
		a:          a,
		b:          b,
		c:          c,
		abDistance: newDistanceQuery(a.LatLng, b.LatLng),
		acDistance: newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance: newDistanceQuery(b.LatLng, c.LatLng),
	}
}
func (s *avoidThinTrianglesScorer) setTriangleScore(a, b, c portalData, score float32) {
	s.minHeight[a.Index][b.Index][c.Index] = score
	s.minHeight[a.Index][c.Index][b.Index] = score
	s.minHeight[b.Index][a.Index][c.Index] = score
	s.minHeight[b.Index][c.Index][a.Index] = score
	s.minHeight[c.Index][a.Index][b.Index] = score
	s.minHeight[c.Index][b.Index][a.Index] = score
}
func (s *avoidThinTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minHeight[a.Index][b.Index][c.Index]
}
func (s *avoidThinTrianglesTriangleScorer) scoreFirstLevelTriangle(p portalData) float32 {
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
func (s *avoidThinTrianglesTriangleScorer) scoreHighLevelTriangle(p portalData) float32 {
	return float32Min(
		s.minHeight[p.Index][s.a.Index][s.b.Index],
		float32Min(
			s.minHeight[p.Index][s.a.Index][s.c.Index],
			s.minHeight[p.Index][s.b.Index][s.c.Index]))
}

func (s *avoidSmallTrianglesScorer) newTriangleScorer(a, b, c portalData) homogeneousTriangleScorer {
	return &avoidSmallTrianglesTriangleScorer{
		minArea: s.minArea,
		a:       a,
		b:       b,
		c:       c,
	}
}
func (s *avoidSmallTrianglesScorer) setTriangleScore(a, b, c portalData, score float32) {
	s.minArea[a.Index][b.Index][c.Index] = score
	s.minArea[a.Index][c.Index][b.Index] = score
	s.minArea[b.Index][a.Index][c.Index] = score
	s.minArea[b.Index][c.Index][a.Index] = score
	s.minArea[c.Index][a.Index][b.Index] = score
	s.minArea[c.Index][b.Index][a.Index] = score
}
func (s *avoidSmallTrianglesScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.minArea[a.Index][b.Index][c.Index]
}
func (s *avoidSmallTrianglesTriangleScorer) scoreFirstLevelTriangle(p portalData) float32 {
	return float32(
		float64Min(
			triangleArea(s.a, s.b, p),
			float64Min(triangleArea(s.a, s.c, p), triangleArea(s.b, s.c, p))) * unitAreaToSquareMeters)
}
func (s *avoidSmallTrianglesTriangleScorer) scoreHighLevelTriangle(p portalData) float32 {
	return float32Min(
		s.minArea[p.Index][s.a.Index][s.b.Index],
		float32Min(
			s.minArea[p.Index][s.a.Index][s.c.Index],
			s.minArea[p.Index][s.b.Index][s.c.Index]))
}
