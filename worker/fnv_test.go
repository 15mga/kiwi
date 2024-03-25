package worker

import (
	"github.com/15mga/kiwi/util"
	"math"
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	c := 9999999
	strSlc := make([]string, c)
	bytesSlc := make([][]byte, c)
	for i := 0; i < len(strSlc); i++ {
		strSlc[i] = strconv.Itoa(i)
		bytesSlc[i] = util.Int64ToBytes(int64(i))
	}
	fnvStrArr := [16]int{}
	memStrArr := [16]int{}
	fnvBytesArr := [16]int{}
	memBytesArr := [16]int{}
	for _, s := range strSlc {
		fnvStrArr[FnvHashStr(s)&15]++
		memStrArr[MemHashStr(s)&15]++
	}
	for _, s := range bytesSlc {
		fnvBytesArr[FnvBytes(s)&15]++
		memBytesArr[MemHashBytes(s)&15]++
	}
	t.Log("avg", c/16)
	mi, ma := getMinMax(fnvStrArr)
	t.Log("fnv string", mi, ma, ma-mi)
	mi, ma = getMinMax(memStrArr)
	t.Log("mem string", mi, ma, ma-mi)
	mi, ma = getMinMax(fnvBytesArr)
	t.Log("fnv bytes", mi, ma, ma-mi)
	mi, ma = getMinMax(memBytesArr)
	t.Log("mem bytes", mi, ma, ma-mi)
}

func getMinMax(a [16]int) (min, max int) {
	min = math.MaxInt
	max = math.MinInt
	for _, c := range a {
		if c < min {
			min = c
		} else if c > max {
			max = c
		}
	}
	return
}

func BenchmarkFnv(b *testing.B) {
	str := "player_1"
	bytes := []byte(str)
	b.Run("fnv string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			FnvHashStr(str)
		}
	})
	b.Run("mem string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MemHashStr(str)
		}
	})
	b.Run("fnv bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			FnvBytes(bytes)
		}
	})
	b.Run("mem bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MemHashBytes(bytes)
		}
	})
}
