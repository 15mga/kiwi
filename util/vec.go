package util

import (
	"math"
	"math/rand"
)

func Vec2One() Vec2 {
	return Vec2{1, 1}
}

type Vec2 struct {
	X float32
	Y float32
}

func Vec2Equal(a, b Vec2) bool {
	return Cmp(a.X, b.X) == 0 && Cmp(a.Y, b.Y) == 0
}

func Vec2Normalize(v Vec2) Vec2 {
	return Vec2Div(v, Vec2Magnitude(v))
}

func Vec2Radian(a, b Vec2) float32 {
	return float32(math.Acos(float64(Vec2Dot(a, b) / (Vec2Magnitude(a) * Vec2Magnitude(b)))))
}

func Vec2ToRadian(a Vec2) float32 {
	return float32(math.Atan2(float64(a.Y), float64(a.X)))
}

// Vec2Cross 叉积
func Vec2Cross(a, b Vec2) float32 {
	return a.X*b.Y - b.X*a.Y
}

// Vec2Dot 点积
func Vec2Dot(a, b Vec2) float32 {
	return a.X*b.X + a.Y*b.Y
}

func Vec2Add(a, b Vec2) Vec2 {
	return Vec2{X: a.X + b.X, Y: a.Y + b.Y}
}

func Vec2Sub(a, b Vec2) Vec2 {
	return Vec2{X: a.X - b.X, Y: a.Y - b.Y}
}

func Vec2Div(a Vec2, v float32) Vec2 {
	return Vec2{X: a.X / v, Y: a.Y / v}
}

func Vec2Mul(a Vec2, v float32) Vec2 {
	return Vec2{X: a.X * v, Y: a.Y * v}
}

func Vec2Magnitude(a Vec2) float32 {
	return float32(math.Sqrt(float64(Vec2Dot(a, a))))
}

func Vec2Dist(a, b Vec2) float32 {
	return Vec2Magnitude(Vec2Sub(a, b))
}

func Vec2DistSquare(a, b Vec2) float32 {
	v := Vec2Sub(a, b)
	return Vec2Dot(v, v)
}

func Vec2Clamp(v, min, max Vec2) Vec2 {
	return Vec2{
		X: ClampFloat(v.X, min.X, max.X),
		Y: ClampFloat(v.Y, min.Y, max.Y),
	}
}

func Vec2Rotate90(a Vec2) Vec2 {
	return Vec2{
		X: -a.Y,
		Y: a.X,
	}
}

func Vec2Rotate180(v Vec2) Vec2 {
	return Vec2{
		X: -v.X,
		Y: -v.Y,
	}
}

func Vec2Rotate270(a Vec2) Vec2 {
	return Vec2{
		X: a.Y,
		Y: -a.X,
	}
}

func Vec2Rotate(a Vec2, radian float32) Vec2 {
	r := float64(radian)
	s := math.Sin(r)
	c := math.Cos(r)
	return Vec2{
		X: float32(float64(a.X)*c - float64(a.Y)*s),
		Y: float32(float64(a.X)*s + float64(a.Y)*c),
	}
}

func Vec2Lerp(a, b Vec2, v float32) Vec2 {
	return Vec2{
		X: a.X + (b.X-a.X)*v,
		Y: a.Y + (b.Y-a.Y)*v,
	}
}

func Vec2XRebound(a, b Vec2, bound float32) (Vec2, bool) {
	if (a.X < bound && b.X > bound) ||
		(a.X > bound && b.X < bound) {
		return Vec2{}, false
	}
	a1X := bound - a.X + bound
	u := (bound - b.X) / (a1X - b.X)
	w := b.Y + (a.Y-b.Y)*u
	return Vec2{X: bound, Y: w}, true
}

func Vec2YRebound(a, b Vec2, bound float32) (Vec2, bool) {
	if (a.Y < bound && b.Y > bound) ||
		(a.Y > bound && b.Y < bound) {
		return Vec2{}, false
	}
	a1Y := bound - a.Y + bound
	u := (bound - b.Y) / (a1Y - b.Y)
	w := b.X + (a.X-b.X)*u
	return Vec2{X: w, Y: bound}, true
}

func PointToVecNearPoint(p, a, b Vec2) Vec2 {
	dot := (b.X-a.X)*(p.X-a.X) + (b.Y-a.Y)*(p.Y-a.Y)
	dotP := (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y)
	u := dot / dotP
	x := a.X + (b.X-a.X)*u
	y := a.Y + (b.Y-a.Y)*u
	return Vec2{
		X: x,
		Y: y,
	}
}

func PointToVecDistSquare(p, a, b Vec2) float32 {
	dot := (b.X-a.X)*(p.X-a.X) + (b.Y-a.Y)*(p.Y-a.Y)
	dotP := (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y)
	u := dot / dotP
	x := a.X + (b.X-a.X)*u
	y := a.Y + (b.Y-a.Y)*u
	x2 := p.X - x
	y2 := p.Y - y
	return x2*x2 + y2*y2
}

func PointToVecDist(p, a, b Vec2) float32 {
	return float32(math.Sqrt(float64(PointToVecDistSquare(p, a, b))))
}

func PointToSegNearPoint(p, a, b Vec2) Vec2 {
	dot := (b.X-a.X)*(p.X-a.X) + (b.Y-a.Y)*(p.Y-a.Y)
	if CmpZero(dot) < 1 {
		return a
	}
	dotP := (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y)
	if Cmp(dot, dotP) > -1 {
		return b
	}
	u := dot / dotP
	x := a.X + (b.X-a.X)*u
	y := a.Y + (b.Y-a.Y)*u
	return Vec2{
		X: x,
		Y: y,
	}
}

func PointToSegDistSquare(p, a, b Vec2) float32 {
	dot := (b.X-a.X)*(p.X-a.X) + (b.Y-a.Y)*(p.Y-a.Y)
	if dot <= 0 {
		return (a.X-p.X)*(a.X-p.X) + (a.Y-p.Y)*(a.Y-p.Y)
	}
	dotP := (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y)
	if dot >= dotP {
		return (b.X-p.X)*(b.X-p.X) + (b.Y-p.Y)*(b.Y-p.Y)
	}
	u := dot / dotP
	x := a.X + (b.X-a.X)*u
	y := a.Y + (b.Y-a.Y)*u
	x2 := p.X - x
	y2 := p.Y - y
	return x2*x2 + y2*y2
}

// PointToSegDist 点到线段的距离
func PointToSegDist(p, a, b Vec2) float32 {
	return float32(math.Sqrt(float64(PointToSegDistSquare(p, a, b))))
}

// TriangleArea 三点构成的三角形面积
func TriangleArea(a, b, c Vec2) float32 {
	return Vec2Cross(Vec2Sub(b, a), Vec2Sub(c, a))
}

// PointOnSeg 点是否在线段上
func PointOnSeg(p, a, b Vec2) bool {
	v1 := Vec2Sub(a, p)
	v2 := Vec2Sub(b, p)
	return CmpZero(Vec2Cross(v1, v2)) == 0 && CmpZero(Vec2Dot(v1, v2)) == -1
}

func PointInTriangle(p, a, b, c Vec2) bool {
	pa := Vec2Sub(a, p)
	pb := Vec2Sub(b, p)
	pc := Vec2Sub(c, p)
	c0 := Vec2Cross(pa, pb)
	c1 := Vec2Cross(pb, pc)
	c2 := Vec2Cross(pc, pa)
	return (c0 < 0 && c1 < 0 && c2 < 0) ||
		(c0 > 0 && c1 > 0 && c2 > 0)
}

func Vec3One() Vec3 {
	return Vec3{1, 1, 1}
}

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

type Vec2Int struct {
	X int32
	Y int32
}

func (v Vec2Int) Equal(v1 Vec2Int) bool {
	return v.X == v1.X && v.Y == v1.Y
}

func RandDir() Vec2 {
	x := rand.Float32()*2 - 1
	y := rand.Float32()*2 - 1
	return Vec2Normalize(Vec2{
		X: x,
		Y: y,
	})
}

func RandRange(center Vec2, radius float32) Vec2 {
	return Vec2Add(center, Vec2Mul(RandDir(), radius))
}
