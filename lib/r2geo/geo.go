package r2geo

import "math"

import "github.com/golang/geo/r2"

// TriangleQuery helps to answer question whethen a point is contained
// inside triangle.
type TriangleQuery struct {
	a, b, c r2.Point
}

// Sign returns a positive number if points a,b,c are ordered counterclockwise,
// and negative number if they are ordered clockwise.
func Sign(a, b, c r2.Point) float64 {
	return (a.X-c.X)*(b.Y-c.Y) - (b.X-c.X)*(a.Y-c.Y)
}

func NewTriangleQuery(a, b, c r2.Point) TriangleQuery {
	if Sign(a, b, c) < 0 {
		a, c = c, a
	}
	return TriangleQuery{a, b, c}
}

func (t *TriangleQuery) ContainsPoint(o r2.Point) bool {
	if Sign(t.a, t.b, o) > 0 && Sign(t.c, t.a, o) > 0 && Sign(t.b, t.c, o) > 0 {
		return true
	}
	return false
}

// orderedCCWQuery helps to answer question whether semiline ob
// lies between semilines oa and oc (looking in counterclockwise order).
type orderedCCWQuery struct {
	a, o, c r2.Point
	aocSign bool
}

func newOrderedCCWQuery(a, c, o r2.Point) orderedCCWQuery {
	aocSign := Sign(a, o, c) > 0
	return orderedCCWQuery{a, o, c, aocSign}
}

func (t *orderedCCWQuery) Ordered(b r2.Point) bool {
	if t.aocSign {
		return Sign(b, t.o, t.a) > 0 || Sign(t.c, t.o, b) > 0
	}
	return Sign(b, t.o, t.a) > 0 && Sign(t.c, t.o, b) > 0
}

// triangleWedgeQuery helps to answer question wether a point is contained
// inside a wedge which is a contained between semilines ab and ac, where angle
// between ab and ac is < pi.
type TriangleWedgeQuery struct {
	ccwQuery orderedCCWQuery
}

func NewTriangleWedgeQuery(a, b, c r2.Point) TriangleWedgeQuery {
	if Sign(a, b, c) > 0 {
		return TriangleWedgeQuery{newOrderedCCWQuery(b, c, a)}
	}
	return TriangleWedgeQuery{newOrderedCCWQuery(c, b, a)}
}

func (t *TriangleWedgeQuery) ContainsPoint(o r2.Point) bool {
	return t.ccwQuery.Ordered(o)
}

// DistanceQuery helps find distance from segment a,b to a point
type DistanceQuery struct {
	a, b     r2.Point
	ab       r2.Point
	invLenSq float64
}

func NewDistanceQuery(a, b r2.Point) DistanceQuery {
	ab := b.Sub(a)
	lenSq := ab.X*ab.X + ab.Y*ab.Y
	if lenSq == 0 {
		return DistanceQuery{
			a:        a,
			b:        b,
			ab:       b.Sub(a),
			invLenSq: 0,
		}
	}
	return DistanceQuery{
		a:        a,
		b:        b,
		ab:       b.Sub(a),
		invLenSq: 1. / lenSq,
	}
}
func length(p r2.Point) float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}
func lengthSq(p r2.Point) float64 {
	return p.X*p.X + p.Y*p.Y
}
func (d *DistanceQuery) Distance(p r2.Point) float64 {
	if d.invLenSq == 0 {
		return length(p.Sub(d.a))
	}
	t := p.Sub(d.a).Dot(d.ab) * d.invLenSq
	if t <= 0 {
		return length(p.Sub(d.a))
	} else if t >= 1 {
		return length(p.Sub(d.b))
	} else {
		proj := d.a.Add(d.ab.Mul(t))
		return length(p.Sub(proj))
	}
}
func (d *DistanceQuery) DistanceSq(p r2.Point) float64 {
	if d.invLenSq == 0 {
		return lengthSq(p.Sub(d.a))
	}
	t := p.Sub(d.a).Dot(d.ab) * d.invLenSq
	if t <= 0 {
		return lengthSq(p.Sub(d.a))
	} else if t >= 1 {
		return lengthSq(p.Sub(d.b))
	} else {
		proj := d.a.Add(d.ab.Mul(t))
		return lengthSq(p.Sub(proj))
	}
}

// AngleQuery helps find internal angle of triangle abc at vertex b.
type AngleQuery struct {
	b     r2.Point
	ab    r2.Point
	abLen float64
}

func NewAngleQuery(a, b r2.Point) AngleQuery {
	ab := a.Sub(b)
	return AngleQuery{
		b:     b,
		ab:    ab,
		abLen: math.Sqrt(ab.X*ab.X + ab.Y*ab.Y),
	}
}
func (a *AngleQuery) Angle(c r2.Point) float64 {
	bc := c.Sub(a.b)
	bcLen := math.Sqrt(bc.X*bc.X + bc.Y*bc.Y)
	return math.Acos(a.ab.Dot(bc) / (a.abLen * bcLen))
}

// equivalent to math.Atan(a0.Y,a0.X) < math.Atan(a1.Y, a1.X)
func angleLess(a0, a1 r2.Point) bool {
	if (a0.Y < 0) != (a1.Y < 0) {
		return a0.Y < 0
	}
	return a0.Y*a1.X < a1.Y*a0.X
}


func TriangleArea(p0, p1, p2 r2.Point) float64 {
	return math.Abs((p0.X*(p1.Y-p2.Y) + p1.X*(p2.Y-p0.Y) + p2.X*(p0.Y-p1.Y)) * 0.5)
}

func Distance(p0, p1 r2.Point) float64 {
	dx, dy := p0.X-p1.X, p0.Y-p1.Y
	return math.Sqrt(dx*dx + dy*dy)
}
func DistanceSq(p0, p1 r2.Point) float64 {
	dx, dy := p0.X-p1.X, p0.Y-p1.Y
	return dx*dx + dy*dy
}
