package gl

import "github.com/go-gl/mathgl/mgl32"

func rotate90CW(v mgl32.Vec2) mgl32.Vec2 {
	return mgl32.Vec2{v.Y(), -v.X()}
}

func rotate90CCW(v mgl32.Vec2) mgl32.Vec2 {
	return mgl32.Vec2{-v.Y(), v.X()}
}

func cross(v1, v2 mgl32.Vec2) float32 {
	return v1.X()*v2.Y() - v1.Y()*v2.X()
}

func segmentIntersection(v1, v2, v3, v4 mgl32.Vec2) float32 {
	return cross(v1.Sub(v3), v3.Sub(v4)) / cross(v1.Sub(v2), v3.Sub(v4))
}
