package util

import "math"

const (
	Rad2Deg float32 = 180 / math.Pi
	Deg2Rad float32 = math.Pi / 180
	HalfPi  float32 = math.Pi / 2
	Eps     float32 = 1e-5
)

func RadianToDegree(radian float32) float32 {
	return radian * Rad2Deg
}

func DegreeToRadian(angle float32) float32 {
	return angle * Deg2Rad
}

func Round(d float32, bit int32) float32 {
	var v = math.Pow(10, float64(bit))
	return float32(math.Round(float64(d)*v) / v)
}

func Ceil(d float32, bit int32) float32 {
	var v = math.Pow(10, float64(bit))
	return float32(math.Ceil(float64(d)*v) / v)
}

func Floor(d float32, bit int32) float32 {
	var v = math.Pow(10, float64(bit))
	return float32(math.Floor(float64(d)*v) / v)
}

func ClampInt32(min, max, v int32) int32 {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func Clamp(min, max, v float32) float32 {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func FloorInt32(floor, v int32) (r, o int32) {
	if v < floor {
		r = floor
		o = floor - v
	} else {
		r = v
	}
	return
}

func CeilInt32(ceil, v int32) (r, o int32) {
	if v > ceil {
		r = ceil
		o = v - ceil
	} else {
		r = v
	}
	return
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MinInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func MinUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func MaxInt32(a, b int32) int32 {
	if a < b {
		return b
	}
	return a
}

func MaxUint32(a, b uint32) uint32 {
	if a < b {
		return b
	}
	return a
}

func Min(l, r float32) float32 {
	if l > r {
		return r
	} else {
		return l
	}
}

func Max(l, r float32) float32 {
	if l > r {
		return l
	} else {
		return r
	}
}

func Abs(d float32) float32 {
	return math.Float32frombits(math.Float32bits(d) &^ (1 << 31))
}

func IsPowerOfTwo(mask int32) bool {
	return (mask & (mask - 1)) == 0
}

func Sqrt(v float32) float32 {
	return float32(math.Sqrt(float64(v)))
}

func NextPowerOfTwo(v int) int {
	v -= 1
	v |= v >> 16
	v |= v >> 8
	v |= v >> 4
	v |= v >> 2
	v |= v >> 1
	return v + 1
}

func NextCap(required, current, slow int) (int, bool) {
	if required <= current {
		return current, false
	}
	if current < slow {
		return NextPowerOfTwo(required), true
	}
	for current < required {
		current += slow
	}
	return current, true
}

func Lerp(from, to, t float32) float32 {
	return to*t + from*(1.0-t)
}

func CmpZero(v float32) int {
	if Abs(v) <= Eps {
		return 0
	}
	if v < 0 {
		return -1
	}
	return 1
}

func Cmp(v1, v2 float32) int {
	return CmpZero(v1 - v2)
}

func Equal(v1, v2 float32) bool {
	return Abs(v1-v2) < Eps
}

func ClampFloat(v, min, max float32) float32 {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func LerpFloat(v1, v2, v float32) float32 {
	return v1 + (v2-v1)*v
}

func LerpInt(v1, v2, v, t int) int {
	return v1 + (v2-v1)*v/t
}
