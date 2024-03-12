package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRadian(t *testing.T) {
	v1 := Vec2{3, 0}
	for i := 0; i < 360; i++ {
		d := float32(i)
		if d > 180 {
			d -= 360
		}
		r := DegreeToRadian(d)
		v2 := Vec2Rotate(v1, r)
		r1 := Vec2Radian(v1, v2)
		r2 := Abs(Vec2ToRadian(v2))
		assert.Truef(t, Equal(r1, r2), "r1:%f r2:%f", r1, r2)
	}
}

func TestRadian2(t *testing.T) {
	v1 := Vec2{3, 0}
	for i := 0; i < 4; i++ {
		r := DegreeToRadian(float32(i) * 90)
		v2 := Vec2Rotate(v1, r)
		r2 := Vec2Radian(v1, v2)
		fmt.Printf("i:%d r:%f\n", i, r2)
	}
}

func BenchmarkLower(b *testing.B) {
	str := "2benchmark_2lower"
	s := ToBigHump(str)
	fmt.Println(s)
	b.Run("2", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ToBigHump(str)
		}
	})
	b.Run("3", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ToUnderline(s)
		}
	})
}
