package main

import "math"
import "github.com/golang/geo/r2"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

type WebMercatorProjection struct{}

func NewWebMercatorProjection() s2.Projection {
	return &WebMercatorProjection{}
}

// Project converts a point on the sphere to a projected 2D point.
func (p *WebMercatorProjection) Project(pt s2.Point) r2.Point {
	return p.FromLatLng(s2.LatLngFromPoint(pt))
}

// Unproject converts a projected 2D point to a point on the sphere.
func (p *WebMercatorProjection) Unproject(pt r2.Point) s2.Point {
	return s2.PointFromLatLng(p.ToLatLng(pt))
}

// FromLatLng returns the LatLng projected into an R2 Point.
func (p *WebMercatorProjection) FromLatLng(ll s2.LatLng) r2.Point {
	y := (1 - math.Asinh(math.Tan(float64(ll.Lat)))/math.Pi) / 2
	return r2.Point{X: ((float64(ll.Lng) / math.Pi) + 1) / 2, Y: y}
}

// ToLatLng returns the LatLng projected from the given R2 Point.
func (p *WebMercatorProjection) ToLatLng(pt r2.Point) s2.LatLng {
	lat := math.Atan(math.Sinh(math.Pi * (1 - 2*pt.Y)))
	return s2.LatLng{Lat: s1.Angle(lat), Lng: s1.Angle((pt.X*2 - 1) * math.Pi)}
}

// Interpolate returns the point obtained by interpolating the given
// fraction of the distance along the line from A to B.
func (p *WebMercatorProjection) Interpolate(f float64, a, b r2.Point) r2.Point {
	return a.Mul(1 - f).Add(b.Mul(f))
}

// WrapDistance reports the coordinate wrapping distance along each axis.
func (p *WebMercatorProjection) WrapDistance() r2.Point {
	return r2.Point{X: 1, Y: 0}
}
