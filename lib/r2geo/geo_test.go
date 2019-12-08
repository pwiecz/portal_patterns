package r2geo

import "math"
import "testing"

import "github.com/golang/geo/r2"

func TestTriangle(t *testing.T) {
	a := r2.Point{0, 0}
	b := r2.Point{0, 1}
	c := r2.Point{1, 0}
	qPerms := []TriangleQuery{
		NewTriangleQuery(a, b, c),
		NewTriangleQuery(a, c, b),
		NewTriangleQuery(b, a, c),
		NewTriangleQuery(b, c, a),
		NewTriangleQuery(c, a, b),
		NewTriangleQuery(c, b, a)}
	goodPoints := []r2.Point{
		r2.Point{0.1, 0.1},
		r2.Point{0.1, 0.8},
		r2.Point{0.8, 0.1},
		r2.Point{0.4, 0.4},
	}
	badPoints := []r2.Point{
		r2.Point{0.6, 0.6},
		r2.Point{1.0, 1.0},
		r2.Point{-0.1, 0},
		r2.Point{1.0, 0.1},
	}
	for i, q := range qPerms {
		for _, p := range goodPoints {
			if !q.ContainsPoint(p) {
				t.Errorf("Triangle %d should contain point %f,%f", i, p.X, p.Y)
			}
		}
		for _, p := range badPoints {
			if q.ContainsPoint(p) {
				t.Errorf("Triangle %d should not contain point %f,%f", i, p.X, p.Y)
			}
		}
	}
}

func TestWedge(t *testing.T) {
	a := r2.Point{0, 0}
	b := r2.Point{1, 0}
	c := r2.Point{1, 1}
	qPerms := []TriangleWedgeQuery{
		NewTriangleWedgeQuery(a, b, c),
		NewTriangleWedgeQuery(a, c, b),
	}
	goodPoints := []r2.Point{
		r2.Point{0.1, 0.09},
		r2.Point{4, 0.1},
		r2.Point{4, 3.9},
	}
	badPoints := []r2.Point{
		r2.Point{0.1, 0.2},
		r2.Point{-0.1, -0.1},
		r2.Point{0.1, -0.1},
	}
	for i, q := range qPerms {
		for _, p := range goodPoints {
			if !q.ContainsPoint(p) {
				t.Errorf("Wedge %d should contain point %f,%f", i, p.X, p.Y)
			}
		}
		for _, p := range badPoints {
			if q.ContainsPoint(p) {
				t.Errorf("Wedge %d should not contain point %f,%f", i, p.X, p.Y)
			}
		}
	}
}

func TestDistance(t *testing.T) {
	a := r2.Point{0, 0}
	b := r2.Point{1,0}
	q:= NewDistanceQuery(a, b)
	points := []r2.Point{
		r2.Point{-0.5, 0},
		r2.Point{0, 0.25},
		r2.Point{0.1, 0.125},
		r2.Point{1, 1},
		r2.Point{2, -1},
	}
	results := []float64 {
		0.5,
		0.25,
		0.125,
		1,
		math.Sqrt(2),
	}
	for i, point := range points {
		d := q.Distance(point)
		if math.Abs(d - results[i]) > 0.00001 {
			t.Errorf("Expected distance for point %f,%f - %f got %f", point.X, point.Y, results[i], d)
		}
	}
}
