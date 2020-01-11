package lib

import "math"

import "github.com/golang/geo/r3"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

const radiansToMeters = 2e+7 / math.Pi
const unitAreaToSquareMeters = 5.1e+14

// ccwQuery helps answer question whether three points a, b, p are counterclockwise
type ccwQuery r3.Vector

func newCCWQuery(a, b s2.Point) ccwQuery {
	return ccwQuery(a.Cross(b.Vector))
}

func (c ccwQuery) IsCCW(p s2.Point) bool {
	return r3.Vector(c).Dot(p.Vector) > 0
}

// triangleQuery helps to answer question whethen a point is contained
// inside triangle.
type triangleQuery struct {
	aCrossB, cCrossA, bCrossC r3.Vector
}

func newTriangleQuery(a, b, c s2.Point) triangleQuery {
	if !s2.Sign(a, b, c) {
		a, c = c, a
	}
	aCrossB := a.Cross(b.Vector)
	cCrossA := c.Cross(a.Vector)
	bCrossC := b.Cross(c.Vector)
	return triangleQuery{aCrossB, cCrossA, bCrossC}
}

// returns true if points abc are counterclockwise
func sign(aCrossB r3.Vector, c s2.Point) bool {
	return aCrossB.Dot(c.Vector) > 0
}

func (t *triangleQuery) ContainsPoint(o s2.Point) bool {
	if sign(t.aCrossB, o) && sign(t.cCrossA, o) && sign(t.bCrossC, o) {
		return true
	}
	return false
}

// orderedCCWQuery helps to answer question whether semiline ob
// lies between semilines oa and oc (looking in counter clockwise order).
type orderedCCWQuery struct {
	a, o    s2.Point
	aocSign bool
	cCrossO r3.Vector
}

func newOrderedCCWQuery(a, c, o s2.Point) orderedCCWQuery {
	aCrossO := a.Cross(o.Vector)
	aocSign := sign(aCrossO, c)
	cCrossO := c.Cross(o.Vector)
	return orderedCCWQuery{a, o, aocSign, cCrossO}
}

func (t *orderedCCWQuery) Ordered(b s2.Point) bool {
	if t.aocSign {
		return sign(b.Cross(t.o.Vector), t.a) ||
			sign(t.cCrossO, b)
	}
	return sign(b.Cross(t.o.Vector), t.a) &&
		sign(t.cCrossO, b)
}

// triangleWedgeQuery helps to answer question wether a point is contained
// inside a wedge which is a contained between semilines ab and ac, where angle
// between ab and ac is < pi.
type triangleWedgeQuery struct {
	ccwQuery orderedCCWQuery
}

func newTriangleWedgeQuery(a, b, c s2.Point) triangleWedgeQuery {
	if sign(a.Cross(b.Vector), c) {
		return triangleWedgeQuery{newOrderedCCWQuery(b, c, a)}
	}
	return triangleWedgeQuery{newOrderedCCWQuery(c, b, a)}
}

func (t *triangleWedgeQuery) ContainsPoint(o s2.Point) bool {
	return t.ccwQuery.Ordered(o)
}

// distanceQuery helps find distance from segment a,b to a point
type distanceQuery struct {
	aCrossB s2.Point
	invC2   float64
}

func newDistanceQuery(a, b s2.Point) distanceQuery {
	aCrossB := a.PointCross(b)
	return distanceQuery{aCrossB, 1.0 / aCrossB.Norm2()}
}
func (d *distanceQuery) ChordAngle(p s2.Point) s1.ChordAngle {
	pDotC := p.Dot(d.aCrossB.Vector)
	pDotC2 := pDotC * pDotC
	cx := d.aCrossB.Cross(p.Vector)
	qr := 1 - math.Sqrt(cx.Norm2()*d.invC2)
	return s1.ChordAngle((pDotC2 * d.invC2) + (qr * qr))
}
func (d *distanceQuery) Distance(p s2.Point) s1.Angle {
	return d.ChordAngle(p).Angle()
}

func triangleArea(p0, p1, p2 portalData) float64 {
	return s2.GirardArea(p0.LatLng, p1.LatLng, p2.LatLng)
}

func distance(p0, p1 portalData) float64 {
	return p0.LatLng.Sub(p1.LatLng.Vector).Norm()
}
func distanceSq(p0, p1 portalData) float64 {
	return p0.LatLng.Sub(p1.LatLng.Vector).Norm2()
}

type AngleQuery struct {
	a  s2.Point
	ab r3.Vector
}

func NewAngleQuery(a, b s2.Point) AngleQuery {
	return AngleQuery{
		a:  a,
		ab: b.PointCross(a).Vector,
	}
}
func (a *AngleQuery) Angle(c s2.Point) s1.Angle {
	return c.PointCross(a.a).Angle(a.ab)
}
