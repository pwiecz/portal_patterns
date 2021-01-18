package lib

type thickTrianglesPureScorer struct{}

func (s thickTrianglesPureScorer) scoreTrianglePure(a, b, c portalData, level int, portals []portalData) float32 {
	if level <= 1 {
		return 0
	}

	portalsInTriangle := portalsInsideTriangle(portals, a, b, c, nil)
	center := findHomogeneousCenterPortal(a, b, c, portalsInTriangle)
	if level == 2 {
		q1 := newDistanceQuery(a.LatLng, b.LatLng)
		abDistance := float64(q1.Distance(center.LatLng))
		q2 := newDistanceQuery(b.LatLng, c.LatLng)
		bcDistance := float64(q2.Distance(center.LatLng))
		q3 := newDistanceQuery(a.LatLng, c.LatLng)
		acDistance := float64(q3.Distance(center.LatLng))
		return float32(
			float64Min(abDistance,
				float64Min(bcDistance, acDistance)) * RadiansToMeters)
	}
	return float32Min(
		s.scoreTrianglePure(a, b, center, level-1, portalsInTriangle),
		float32Min(
			s.scoreTrianglePure(b, c, center, level-1, portalsInTriangle),
			s.scoreTrianglePure(c, a, center, level-1, portalsInTriangle)))
}
