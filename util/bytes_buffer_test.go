package util

import (
	"bytes"
	"strconv"
	"sync"
	"testing"
)

func BenchmarkBytes(b *testing.B) {
	count := 10
	pc := 50000
	b.Run("buffer string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(pc)
			for i := 0; i < pc; i++ {
				go func() {
					var buffer ByteBuffer
					buffer.InitCap(190)
					for i := 0; i < count; i++ {
						buffer.WStringNoLen(strconv.Itoa(i))
					}
					buffer.Dispose()
					wg.Done()
				}()
			}
			wg.Wait()
		}
	})
	b.Run("builder string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(pc)
			for i := 0; i < pc; i++ {
				go func() {
					builder := bytes.NewBuffer(nil)
					for i := 0; i < count; i++ {
						builder.WriteString(strconv.Itoa(i))
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
	})
}
