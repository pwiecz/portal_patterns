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

type arbitraryScorer struct{}

type largestTriangleTopLevelScorer struct{}
type smallestTriangleTopLevelScorer struct{}

func newAvoidThinTrianglesScorer(numPortals int) *avoidThinTrianglesScorer {
	minHeight := make([][][]float32, 0, numPortals)
	for i := 0; i < numPortals; i++ {
		minHeight = append(minHeight, make([][]float32, 0, numPortals))
		for j := 0; j < numPortals; j++ {
			minHeight[i] = append(minHeight[i], make([]float32, numPortals))
		}
	}
	return &avoidThinTrianglesScorer{
		minHeight: minHeight,
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

func (s arbitraryScorer) newTriangleScorer(a, b, c portalData) homogeneousTriangleScorer {
	return arbitraryScorer{}
}
func (s arbitraryScorer) setTriangleScore(a, b, c portalData, score float32) {}
func (s arbitraryScorer) scoreTriangle(a, b, c portalData) float32           { return 0 }
func (s arbitraryScorer) scoreFirstLevelTriangle(p portalData) float32       { return 0 }
func (s arbitraryScorer) scoreHighLevelTriangle(p portalData) float32        { return 0 }

func (s largestTriangleTopLevelScorer) scoreTriangle(a, b, c portalData) float32 {
	return float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}

func (s smallestTriangleTopLevelScorer) scoreTriangle(a, b, c portalData) float32 {
	return -float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
