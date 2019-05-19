package main

import "math"

type avoidThinTriangles2Scorer struct {
	minHeight [][][]float32
}
type minHeightVarianceScorer struct {
	counts           [][][]uint16
	sumHeights       [][][]float32
	sumHeightSquares [][][]float32
}

func newAvoidThinTriangles2Scorer(numPortals int) *avoidThinTriangles2Scorer {
	minHeight := make([][][]float32, 0, numPortals)
	for i := 0; i < numPortals; i++ {
		minHeight = append(minHeight, make([][]float32, 0, numPortals))
		for j := 0; j < numPortals; j++ {
			minHeight[i] = append(minHeight[i], make([]float32, numPortals))
		}
	}
	return &avoidThinTriangles2Scorer{
		minHeight: minHeight,
	}
}

func newMinHeightVarianceScorer(numPortals int) *minHeightVarianceScorer {
	counts := make([][][]uint16, 0, numPortals)
	sumHeights := make([][][]float32, 0, numPortals)
	sumHeightSquares := make([][][]float32, 0, numPortals)
	for i := 0; i < numPortals; i++ {
		counts = append(counts, make([][]uint16, 0, numPortals))
		sumHeights = append(sumHeights, make([][]float32, 0, numPortals))
		sumHeightSquares = append(sumHeightSquares, make([][]float32, 0, numPortals))
		for j := 0; j < numPortals; j++ {
			counts[i] = append(counts[i], make([]uint16, numPortals))
			sumHeights[i] = append(sumHeights[i], make([]float32, numPortals))
			sumHeightSquares[i] = append(sumHeightSquares[i], make([]float32, numPortals))
		}
	}
	return &minHeightVarianceScorer{
		counts:           counts,
		sumHeights:       sumHeights,
		sumHeightSquares: sumHeightSquares,
	}
}

type avoidThinTriangles2TriangleScorer struct {
	minHeight  [][][]float32
	maxDepth   int
	a, b, c    portalData
	abDistance distanceQuery
	acDistance distanceQuery
	bcDistance distanceQuery
	scorePtrs  [6]*float32
	candidates [6]uint16
}

func (s *avoidThinTriangles2Scorer) newTriangleScorer(a, b, c portalData, maxDepth int) homogeneous2TriangleScorer {
	a, b, c = sorted(a, b, c)
	var scorePtrs [6]*float32
	for level := 0; level < maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		scorePtrs[level] = &s.minHeight[i][j][k]
	}
	candidates := [6]uint16{
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1}
	return &avoidThinTriangles2TriangleScorer{
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
func (s *avoidThinTriangles2TriangleScorer) scoreCandidate(p portalData) {
	for level := 0; level < s.maxDepth; level++ {
		var minHeight float32
		if level == 0 {
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
			ss0, ss1, ss2 := indexOrdering(s0, s1, s2, level-1)
			t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
			st0, st1, st2 := indexOrdering(t0, t1, t2, level-1)
			u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
			su0, su1, su2 := indexOrdering(u0, u1, u2, level-1)
			minHeight = float32Min(
				s.minHeight[ss0][ss1][ss2],
				float32Min(
					s.minHeight[st0][st1][st2],
					s.minHeight[su0][su1][su2]))
		}
		if minHeight == 0 {
			break
		}
		if minHeight > *s.scorePtrs[level] {
			*s.scorePtrs[level] = minHeight
			s.candidates[level] = p.Index
		}
	}
}

func (s *avoidThinTriangles2TriangleScorer) bestMidpoints() [6]uint16 {
	return s.candidates
}

type minHeightVarianceTriangleScorer struct {
	counts              [][][]uint16
	sumHeights          [][][]float32
	sumHeightSquares    [][][]float32
	maxDepth            int
	a, b, c             portalData
	abDistance          distanceQuery
	acDistance          distanceQuery
	bcDistance          distanceQuery
	countPtrs           [6]*uint16
	sumHeightPtrs       [6]*float32
	sumHeightSquarePtrs [6]*float32
	variances           [6]float32
	candidates          [6]uint16
}

func (s *minHeightVarianceScorer) newTriangleScorer(a, b, c portalData, maxDepth int) homogeneous2TriangleScorer {
	a, b, c = sorted(a, b, c)
	var countPtrs [6]*uint16
	var sumHeightPtrs [6]*float32
	var sumHeightSquarePtrs [6]*float32
	for level := 0; level < maxDepth; level++ {
		i, j, k := indexOrdering(a.Index, b.Index, c.Index, level)
		countPtrs[level] = &s.counts[i][j][k]
		sumHeightPtrs[level] = &s.sumHeights[i][j][k]
		sumHeightSquarePtrs[level] = &s.sumHeightSquares[i][j][k]
	}
	variances := [6]float32{
		-math.MaxFloat32,
		-math.MaxFloat32,
		-math.MaxFloat32,
		-math.MaxFloat32,
		-math.MaxFloat32,
		-math.MaxFloat32,
	}
	candidates := [6]uint16{
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1}
	return &minHeightVarianceTriangleScorer{
		counts:              s.counts,
		sumHeights:          s.sumHeights,
		sumHeightSquares:    s.sumHeightSquares,
		maxDepth:            maxDepth,
		a:                   a,
		b:                   b,
		c:                   c,
		abDistance:          newDistanceQuery(a.LatLng, b.LatLng),
		acDistance:          newDistanceQuery(a.LatLng, c.LatLng),
		bcDistance:          newDistanceQuery(b.LatLng, c.LatLng),
		countPtrs:           countPtrs,
		sumHeightPtrs:       sumHeightPtrs,
		sumHeightSquarePtrs: sumHeightSquarePtrs,
		variances:           variances,
		candidates:          candidates,
	}
}
func (s *minHeightVarianceScorer) scoreTriangle(a, b, c portalData) float32 {
	count := float32(s.counts[a.Index][b.Index][c.Index])
	sumHeights := s.sumHeights[a.Index][b.Index][c.Index]
	sumHeightSquares := s.sumHeightSquares[a.Index][b.Index][c.Index]
	return -(sumHeightSquares - sumHeights*sumHeights/count) / (count - 1)
}
func (s *minHeightVarianceTriangleScorer) scoreCandidate(p portalData) {
	for level := 0; level < s.maxDepth; level++ {
		var count uint16
		var sumHeights float32
		var sumHeightSquares float32
		if level == 0 {
			d1 := float32(s.abDistance.Distance(p.LatLng).Radians() * radiansToMeters)
			d2 := float32(s.acDistance.Distance(p.LatLng).Radians() * radiansToMeters)
			d3 := float32(s.bcDistance.Distance(p.LatLng).Radians() * radiansToMeters)
			count = 3
			sumHeights = d1 + d2 + d3
			sumHeightSquares = d1*d1 + d2*d2 + d3*d3
		} else {
			s0, s1, s2 := merge(p.Index, s.a.Index, s.b.Index)
			ss0, ss1, ss2 := indexOrdering(s0, s1, s2, level-1)
			t0, t1, t2 := merge(p.Index, s.a.Index, s.c.Index)
			st0, st1, st2 := indexOrdering(t0, t1, t2, level-1)
			u0, u1, u2 := merge(p.Index, s.b.Index, s.c.Index)
			su0, su1, su2 := indexOrdering(u0, u1, u2, level-1)
			count0 := s.counts[ss0][ss1][ss2]
			count1 := s.counts[st0][st1][st2]
			count2 := s.counts[su0][su1][su2]
			if count0 == 0 || count1 == 0 || count2 == 0 {
				break
			}
			count = count0 + count1 + count2
			sumHeights = s.sumHeights[ss0][ss1][ss2] + s.sumHeights[st0][st1][st2] + s.sumHeights[su0][su1][su2]
			sumHeightSquares = s.sumHeightSquares[ss0][ss1][ss2] + s.sumHeightSquares[st0][st1][st2] + s.sumHeightSquares[su0][su1][su2]
		}
		fCount := float32(count)
		variance := (sumHeightSquares - sumHeights*sumHeights/fCount) / (fCount - 1)
		if -variance > s.variances[level] {
			s.variances[level] = -variance
			*s.countPtrs[level] = count
			*s.sumHeightPtrs[level] = sumHeights
			*s.sumHeightSquarePtrs[level] = sumHeightSquares
			s.candidates[level] = p.Index
		}
	}
}
func (s *minHeightVarianceTriangleScorer) bestMidpoints() [6]uint16 {
	return s.candidates
}
