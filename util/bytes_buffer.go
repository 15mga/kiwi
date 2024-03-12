package util

import (
	"encoding/gob"
	"encoding/hex"
	"io"
	"math"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/proto"
)

const (
	_MinBytesCap = uint32(16)
	_MaxBytesCap = uint32(1 << 20)
)

type _BT = uint8

const (
	Bool _BT = iota
	Bools
	Bytes
	Uint8
	Int8
	Int8s
	Uint16
	Uint16s
	Int16
	Int16s
	Int
	Ints
	Uint32
	Uint32s
	Int32
	Int32s
	Uint64
	Uint64s
	Int64
	Int64s
	Float32
	Float32s
	Float64
	Float64s
	Vector2
	Vector2s
	Vector3
	Vector3s
	Str
	Strs
	Time
	Times
	MInt32
	MInt64
	MStr
	MAny
	Uint16MUint16
	Int32MInt32
	Int32MInt64
	Int64MInt32
	Int64MInt64
	AnyMAny
)

var (
	_BytesPoolMap map[uint32]*sync.Pool
	_Empty        []byte
)

func init() {
	_BytesPoolMap = make(map[uint32]*sync.Pool)
	_Empty = make([]byte, _MaxBytesCap)
	for i := _MinBytesCap; i < _MaxBytesCap; i = i << 1 {
		l := i
		_BytesPoolMap[i] = &sync.Pool{
			New: func() any {
				return make([]byte, l)
			},
		}
	}
}

func SpawnBytes() []byte {
	return SpawnBytesWithLen(_MinBytesCap)
}

func SpawnBytesWithLen(l uint32) []byte {
	//return make([]byte, l)
	var c uint32
	if l < _MinBytesCap {
		c = _MinBytesCap
	} else {
		c = NextPowerOfTwo(l)
		if c > _MaxBytesCap {
			return make([]byte, l)
		}
	}
	return _BytesPoolMap[c].Get().([]byte)
}

func RecycleBytes(bytes []byte) {
	//return
	c := uint32(cap(bytes))
	if c < _MinBytesCap || c > _MaxBytesCap {
		return
	}
	ll := NextPowerOfTwo(c)
	if c < ll {
		c = ll >> 1
	}
	_BytesPoolMap[c].Put(bytes[:c])
}

func CopyBytes(src []byte) []byte {
	l := len(src)
	dst := SpawnBytesWithLen(uint32(l))
	copy(dst, src)
	return dst[:l]
}

type ByteBuffer struct {
	canRecycle bool
	pos        uint32
	len        uint32
	cap        uint32
	bytes      []byte
}

func (b *ByteBuffer) Reset() {
	b.pos = 0
}

func (b *ByteBuffer) InitCap(c uint32) {
	b.pos = 0
	b.len = 0
	b.bytes = SpawnBytesWithLen(c)
	b.cap = uint32(len(b.bytes))
	b.canRecycle = true
}

func (b *ByteBuffer) InitBytes(bytes []byte) {
	b.pos = 0
	b.len = uint32(len(bytes))
	b.cap = b.len
	b.bytes = bytes
	b.canRecycle = false
}

func (b *ByteBuffer) Pos() uint32 {
	return b.pos
}

func (b *ByteBuffer) SetPos(v uint32) {
	if v < b.len {
		b.pos = v
	} else {
		b.pos = b.len
	}
}

func (b *ByteBuffer) Available() uint32 {
	return b.len - b.pos
}

func (b *ByteBuffer) All() []byte {
	return b.bytes[:b.len]
}

func (b *ByteBuffer) CopyBytes() ([]byte, uint32) {
	return CopyBytes(b.bytes[:b.len]), b.len
}

func (b *ByteBuffer) Length() uint32 {
	return b.len
}

func (b *ByteBuffer) Cap() uint32 {
	return b.cap
}

func (b *ByteBuffer) WBool(v bool) {
	if v {
		b.WUint8(1)
	} else {
		b.WUint8(0)
	}
}

func (b *ByteBuffer) WBools(v []bool) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WBool(d)
	}
}

func (b *ByteBuffer) tryGrow(c uint32) {
	if c <= b.cap {
		return
	}
	c = NextPowerOfTwo(c)
	b.cap = c
	bytes := SpawnBytesWithLen(c)
	copy(bytes, b.bytes[:b.pos])
	RecycleBytes(b.bytes)
	b.bytes = bytes
}

func (b *ByteBuffer) Write(v []byte) (int, error) {
	l := uint32(len(v))
	if v == nil || l == 0 {
		return 0, nil
	}
	return b.write(l, v)
}

func (b *ByteBuffer) write(l uint32, v []byte) (int, error) {
	c := b.pos + l
	b.tryGrow(c)
	l = b.pos
	b.pos = c
	b.len = c
	return copy(b.bytes[l:], v), nil
}

func (b *ByteBuffer) WBytes(v []byte) {
	b.WUint32(uint32(len(v)))
	_, _ = b.Write(v)
}

func (b *ByteBuffer) WUint8(v uint8) {
	c := b.pos + 1
	b.tryGrow(c)
	l := b.pos
	b.pos = c
	b.len = c
	b.bytes[l] = v
}

func (b *ByteBuffer) WUint8s(v []uint8) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WUint8(d)
	}
}

func (b *ByteBuffer) WInt8(v int8) {
	b.WUint8(uint8(v))
}

func (b *ByteBuffer) WInt8s(v []int8) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WInt8(d)
	}
}

func (b *ByteBuffer) WUint16(v uint16) {
	_, _ = b.write(2, []byte{byte(v >> 8), byte(v)})
}

func (b *ByteBuffer) WUint16s(v []uint16) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WUint16(d)
	}
}

func (b *ByteBuffer) WInt16(v int16) {
	b.WUint16(uint16(v))
}

func (b *ByteBuffer) WInt16s(v []int16) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WInt16(d)
	}
}

func (b *ByteBuffer) WInt(v int) {
	_, _ = b.write(4, []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (b *ByteBuffer) WInts(v []int) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WInt(d)
	}
}

func (b *ByteBuffer) WUint32(v uint32) {
	_, _ = b.write(4, []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (b *ByteBuffer) WUint32s(v []uint32) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WUint32(d)
	}
}

func (b *ByteBuffer) WInt32(v int32) {
	b.WUint32(uint32(v))
}

func (b *ByteBuffer) WInt32s(v []int32) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WInt32(d)
	}
}

func (b *ByteBuffer) WUint64(v uint64) {
	_, _ = b.write(8, []byte{byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32), byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (b *ByteBuffer) WUint64s(v []uint64) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WUint64(d)
	}
}

func (b *ByteBuffer) WInt64(v int64) {
	b.WUint64(uint64(v))
}

func (b *ByteBuffer) WInt64s(v []int64) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WInt64(d)
	}
}

func (b *ByteBuffer) WFloat32(v float32) {
	b.WUint32(math.Float32bits(v))
}

func (b *ByteBuffer) WFloat32s(v []float32) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WFloat32(d)
	}
}

func (b *ByteBuffer) WFloat64(v float64) {
	b.WUint64(math.Float64bits(v))
}

func (b *ByteBuffer) WFloat64s(v []float64) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WFloat64(d)
	}
}

func (b *ByteBuffer) WVec2(v Vec2) {
	b.WFloat32(v.X)
	b.WFloat32(v.Y)
}

func (b *ByteBuffer) WVec2s(v []Vec2) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WVec2(d)
	}
}

func (b *ByteBuffer) WVec3(v Vec3) {
	b.WFloat32(v.X)
	b.WFloat32(v.Y)
	b.WFloat32(v.Z)
}

func (b *ByteBuffer) WVec3s(v []Vec3) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WVec3(d)
	}
}

func (b *ByteBuffer) WShortString(v string) {
	l := uint32(len(v))
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	c := b.pos + l
	b.tryGrow(c)
	l = b.pos
	b.pos = c
	b.len = c
	copy(b.bytes[l:], v)
	return
}

func (b *ByteBuffer) WString(v string) {
	l := uint32(len(v))
	b.WUint16(uint16(l))
	if l == 0 {
		return
	}
	c := b.pos + l
	b.tryGrow(c)
	l = b.pos
	b.pos = c
	b.len = c
	copy(b.bytes[l:], v)
	return
}

func (b *ByteBuffer) WStrings(v []string) {
	b.WUint16(uint16(len(v)))
	for _, d := range v {
		b.WString(d)
	}
}

func (b *ByteBuffer) WStringNoLen(v string) {
	if len(v) == 0 {
		return
	}
	_, _ = b.Write([]byte(v))
}

func (b *ByteBuffer) WJson(o any) *Err {
	bytes, err := JsonMarshal(o)
	if err != nil {
		return err
	}
	b.WBytes(bytes)
	return nil
}

func (b *ByteBuffer) errLen(l uint32) *Err {
	return NewErr(EcOutOfRange, M{
		"available": b.len - b.pos,
		"need":      l,
	})
}

func (b *ByteBuffer) RBool() (v bool, err *Err) {
	if b.Available() >= 1 {
		v = b.bytes[b.pos] == byte(1)
		b.pos++
	} else {
		err = b.errLen(1)
	}
	return
}

func (b *ByteBuffer) RBools() (v []bool, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]bool, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RBool()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RBytes() ([]byte, *Err) {
	l, err := b.RUint32()
	if err != nil {
		return nil, err
	}
	if b.Available() < l {
		return nil, b.errLen(l)
	}
	v := CopyBytes(b.bytes[b.pos : b.pos+l])
	b.pos += l
	return v, nil
}

// RAvailable 读取剩余所有字节
func (b *ByteBuffer) RAvailable() (v []byte) {
	v = b.bytes[b.pos:b.len]
	b.pos = b.len
	return
}

func (b *ByteBuffer) Read(p []byte) (n int, err error) {
	if b.pos == b.len {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, b.bytes[b.pos:b.len])
	b.pos = b.len
	return n, nil
}

func (b *ByteBuffer) RUint8() (v uint8, err *Err) {
	if b.Available() >= 1 {
		v = b.bytes[b.pos]
		b.pos++
	} else {
		err = b.errLen(1)
	}
	return
}

func (b *ByteBuffer) RUint8s() (v []uint8, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]uint8, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RUint8()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RInt8() (v int8, err *Err) {
	v1, e := b.RUint8()
	if e == nil {
		v = int8(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RInt8s() (v []int8, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]int8, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RInt8()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RUint16() (v uint16, err *Err) {
	if b.Available() >= 2 {
		v = uint16(b.bytes[b.pos])<<8 | uint16(b.bytes[b.pos+1])
		b.pos += 2
	} else {
		err = b.errLen(2)
	}
	return
}

func (b *ByteBuffer) RUint16s() (v []uint16, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]uint16, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RUint16()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RInt16() (v int16, err *Err) {
	v1, e := b.RUint16()
	if e == nil {
		v = int16(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RInt16s() (v []int16, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]int16, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RInt16()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RInt() (v int, err *Err) {
	if b.Available() >= 4 {
		v = int(b.bytes[b.pos])<<24 | int(b.bytes[b.pos+1])<<16 | int(b.bytes[b.pos+2])<<8 | int(b.bytes[b.pos+3])
		b.pos += 4
	} else {
		err = b.errLen(4)
	}
	return
}

func (b *ByteBuffer) RInts() (v []int, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]int, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RInt()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RUint32() (v uint32, err *Err) {
	if b.Available() >= 4 {
		v = uint32(b.bytes[b.pos])<<24 | uint32(b.bytes[b.pos+1])<<16 | uint32(b.bytes[b.pos+2])<<8 | uint32(b.bytes[b.pos+3])
		b.pos += 4
	} else {
		err = b.errLen(4)
	}
	return
}

func (b *ByteBuffer) RUint32s() (v []uint32, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]uint32, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RUint32()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RInt32() (v int32, err *Err) {
	v1, e := b.RUint32()
	if e == nil {
		v = int32(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RInt32s() (v []int32, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]int32, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RInt32()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RUint64() (v uint64, err *Err) {
	if b.Available() >= 8 {
		v = uint64(b.bytes[b.pos])<<56 | uint64(b.bytes[b.pos+1])<<48 | uint64(b.bytes[b.pos+2])<<40 | uint64(b.bytes[b.pos+3])<<32 | uint64(b.bytes[b.pos+4])<<24 | uint64(b.bytes[b.pos+5])<<16 | uint64(b.bytes[b.pos+6])<<8 | uint64(b.bytes[b.pos+7])
		b.pos += 8
	} else {
		err = b.errLen(8)
	}
	return
}

func (b *ByteBuffer) RUint64s() (v []uint64, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]uint64, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RUint64()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RInt64() (v int64, err *Err) {
	v1, e := b.RUint64()
	if e == nil {
		v = int64(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RInt64s() (v []int64, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]int64, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RInt64()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RFloat32() (v float32, err *Err) {
	v1, e := b.RUint32()
	if e == nil {
		v = math.Float32frombits(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RFloat32s() (v []float32, err *Err) {
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		return
	}
	v = make([]float32, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RFloat32()
		if err != nil {
			return
		}
	}
	return
}

func (b *ByteBuffer) RFloat64() (v float64, err *Err) {
	v1, e := b.RUint64()
	if e == nil {
		v = math.Float64frombits(v1)
	} else {
		err = e
	}
	return
}

func (b *ByteBuffer) RFloat64s() (v []float64, err *Err) {
	p := b.pos
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		b.pos = p
		return
	}
	v = make([]float64, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RFloat64()
		if err != nil {
			b.pos = p
			return
		}
	}
	return
}

func (b *ByteBuffer) RVec2() (v Vec2, err *Err) {
	x, err := b.RFloat32()
	if err != nil {
		return Vec2{}, err
	}
	y, err := b.RFloat32()
	if err != nil {
		return Vec2{}, err
	}
	return Vec2{X: x, Y: y}, nil
}

func (b *ByteBuffer) RVec2s() (v []Vec2, err *Err) {
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		return
	}
	v = make([]Vec2, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RVec2()
		if err != nil {
			return
		}
	}
	return
}

func (b *ByteBuffer) RVec3() (v Vec3, err *Err) {
	x, err := b.RFloat32()
	if err != nil {
		return Vec3{}, err
	}
	y, err := b.RFloat32()
	if err != nil {
		return Vec3{}, err
	}
	z, err := b.RFloat32()
	if err != nil {
		return Vec3{}, err
	}
	return Vec3{X: x, Y: y, Z: z}, nil
}

func (b *ByteBuffer) RVec3s() (v []Vec3, err *Err) {
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		return
	}
	v = make([]Vec3, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RVec3()
		if err != nil {
			return
		}
	}
	return
}

func (b *ByteBuffer) RShortString() (v string, err *Err) {
	var length uint8
	length, err = b.RUint8()
	if err != nil {
		return
	}
	if length == 0 {
		return
	}
	l := uint32(length)
	if b.Available() < l {
		err = b.errLen(l)
		return
	}
	v = string(b.bytes[b.pos : b.pos+l])
	b.pos += l
	return
}

func (b *ByteBuffer) RString() (v string, err *Err) {
	var length uint16
	length, err = b.RUint16()
	if err != nil {
		return
	}
	if length == 0 {
		return
	}
	l := uint32(length)
	if b.Available() < l {
		err = b.errLen(l)
		return
	}
	v = string(b.bytes[b.pos : b.pos+l])
	b.pos += l
	return
}

func (b *ByteBuffer) RStrings() (v []string, err *Err) {
	var l uint16
	l, err = b.RUint16()
	if err != nil {
		return
	}
	v = make([]string, l)
	for i := uint16(0); i < l; i++ {
		v[i], err = b.RString()
		if err != nil {
			return
		}
	}
	return
}

func (b *ByteBuffer) RStringNoLen() (v string) {
	bs := b.RAvailable()
	v = string(bs)
	return
}

func (b *ByteBuffer) RJson(v any) *Err {
	bytes, err := b.RBytes()
	if err != nil {
		return err
	}
	return JsonUnmarshal(bytes, v)
}

func (b *ByteBuffer) ToHex() string {
	return hex.EncodeToString(b.bytes[:b.len])
}

func (b *ByteBuffer) WMInt32(m map[string]int32) {
	b.WUint16(uint16(len(m)))
	for k, v := range m {
		b.WString(k)
		b.WInt32(v)
	}
}

func (b *ByteBuffer) RMInt32() (m map[string]int32, err *Err) {
	var (
		l uint16
		k string
		v int32
	)
	l, err = b.RUint16()
	if err != nil {
		return
	}
	m = make(map[string]int32, l)
	if l == 0 {
		return
	}
	for i := uint16(0); i < l; i++ {
		k, err = b.RString()
		if err != nil {
			return
		}
		v, err = b.RInt32()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WMInt64(m map[string]int64) {
	b.WUint16(uint16(len(m)))
	for k, v := range m {
		b.WString(k)
		b.WInt64(v)
	}
}

func (b *ByteBuffer) RMInt64() (m map[string]int64, err *Err) {
	var (
		l uint16
		k string
		v int64
	)
	l, err = b.RUint16()
	if err != nil {
		return
	}
	m = make(map[string]int64, l)
	if l == 0 {
		return
	}
	for i := uint16(0); i < l; i++ {
		k, err = b.RString()
		if err != nil {
			return
		}
		v, err = b.RInt64()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WUint16MUint16(m map[uint16]uint16) {
	b.WUint16(uint16(len(m)))
	for k, v := range m {
		b.WUint16(k)
		b.WUint16(v)
	}
}

func (b *ByteBuffer) RUint16MUint16() (m map[uint16]uint16, err *Err) {
	var (
		l uint16
		k uint16
		v uint16
	)
	l, err = b.RUint16()
	if err != nil {
		return
	}
	m = make(map[uint16]uint16, l)
	if l == 0 {
		return
	}
	for i := uint16(0); i < l; i++ {
		k, err = b.RUint16()
		if err != nil {
			return
		}
		v, err = b.RUint16()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WInt32MInt32(m map[int32]int32) {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	for k, v := range m {
		b.WInt32(k)
		b.WInt32(v)
	}
}

func (b *ByteBuffer) RInt32MInt32() (m map[int32]int32, err *Err) {
	var (
		l uint8
		k int32
		v int32
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	m = make(map[int32]int32, l)
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RInt32()
		if err != nil {
			return
		}
		v, err = b.RInt32()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WInt32MInt64(m map[int32]int64) {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	for k, v := range m {
		b.WInt32(k)
		b.WInt64(v)
	}
}

func (b *ByteBuffer) RInt32MInt64() (m map[int32]int64, err *Err) {
	var (
		l uint8
		k int32
		v int64
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	m = make(map[int32]int64, l)
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RInt32()
		if err != nil {
			return
		}
		v, err = b.RInt64()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WInt64MInt32(m map[int64]int32) {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	for k, v := range m {
		b.WInt64(k)
		b.WInt32(v)
	}
}

func (b *ByteBuffer) RInt64MInt32() (m map[int64]int32, err *Err) {
	var (
		l uint8
		k int64
		v int32
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	m = make(map[int64]int32, l)
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RInt64()
		if err != nil {
			return
		}
		v, err = b.RInt32()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WInt64MInt64(m map[int64]int64) {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	for k, v := range m {
		b.WInt64(k)
		b.WInt64(v)
	}
}

func (b *ByteBuffer) RInt64MInt64() (m map[int64]int64, err *Err) {
	var (
		l uint8
		k int64
		v int64
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	m = make(map[int64]int64, l)
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RInt64()
		if err != nil {
			return
		}
		v, err = b.RInt64()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WMStr(m MS) {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return
	}
	for k, v := range m {
		b.WString(k)
		b.WString(v)
	}
}

func (b *ByteBuffer) RMStr() (m MS, err *Err) {
	var (
		l uint8
		k string
		v string
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	m = make(MS, l)
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RString()
		if err != nil {
			return
		}
		v, err = b.RString()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WMAny(m M) *Err {
	l := len(m)
	b.WUint16(uint16(l))
	if l == 0 {
		return nil
	}
	for k, v := range m {
		b.WString(k)
		err := b.WAny(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ByteBuffer) RMAny(m M) (err *Err) {
	c, err := b.RUint16()
	if err != nil {
		return err
	}
	for i := uint16(0); i < c; i++ {
		k, err := b.RString()
		if err != nil {
			return err
		}
		v, err := b.RAny()
		if err != nil {
			return err
		}
		m[k] = v
	}
	return nil
}

func (b *ByteBuffer) WAnyMAny(m map[any]any) *Err {
	l := len(m)
	b.WUint8(uint8(l))
	if l == 0 {
		return nil
	}
	for k, v := range m {
		err := b.WAny(k)
		if err != nil {
			return err
		}
		err = b.WAny(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ByteBuffer) RAnyMAny(m map[any]any) (err *Err) {
	var (
		l uint8
		k any
		v any
	)
	l, err = b.RUint8()
	if err != nil {
		return
	}
	if l == 0 {
		return
	}
	for i := uint8(0); i < l; i++ {
		k, err = b.RAny()
		if err != nil {
			return
		}
		v, err = b.RAny()
		if err != nil {
			return
		}
		m[k] = v
	}
	return
}

func (b *ByteBuffer) WAny(v any) *Err {
	switch v := v.(type) {
	case bool:
		b.WUint8(Bool)
		b.WBool(v)
	case []bool:
		b.WUint8(Bools)
		b.WBools(v)
	case uint8:
		b.WUint8(Uint8)
		b.WUint8(v)
	case []byte:
		b.WUint8(Bytes)
		b.WBytes(v)
	case int8:
		b.WUint8(Int8)
		b.WInt8(v)
	case []int8:
		b.WUint8(Int8s)
		b.WInt8s(v)
	case uint16:
		b.WUint8(Uint16)
		b.WUint16(v)
	case []uint16:
		b.WUint8(Uint16s)
		b.WUint16s(v)
	case int16:
		b.WUint8(Int16)
		b.WInt16(v)
	case []int16:
		b.WUint8(Int16s)
		b.WInt16s(v)
	case uint32:
		b.WUint8(Uint32)
		b.WUint32(v)
	case []uint32:
		b.WUint8(Uint32s)
		b.WUint32s(v)
	case int:
		b.WUint8(Int)
		b.WInt(v)
	case []int:
		b.WUint8(Ints)
		b.WInts(v)
	case int32:
		b.WUint8(Int32)
		b.WInt32(v)
	case []int32:
		b.WUint8(Int32s)
		b.WInt32s(v)
	case uint64:
		b.WUint8(Uint64)
		b.WUint64(v)
	case []uint64:
		b.WUint8(Uint64s)
		b.WUint64s(v)
	case int64:
		b.WUint8(Int64)
		b.WInt64(v)
	case []int64:
		b.WUint8(Int64s)
		b.WInt64s(v)
	case float32:
		b.WUint8(Float32)
		b.WFloat32(v)
	case []float32:
		b.WUint8(Float32s)
		b.WFloat32s(v)
	case float64:
		b.WUint8(Float64)
		b.WFloat64(v)
	case []float64:
		b.WUint8(Float64s)
		b.WFloat64s(v)
	case Vec2:
		b.WUint8(Vector2)
		b.WVec2(v)
	case []Vec2:
		b.WUint8(Vector2s)
		b.WVec2s(v)
	case Vec3:
		b.WUint8(Vector3)
		b.WVec3(v)
	case []Vec3:
		b.WUint8(Vector3s)
		b.WVec3s(v)
	case string:
		b.WUint8(Str)
		b.WString(v)
	case []string:
		b.WUint8(Strs)
		b.WStrings(v)
	case time.Time:
		b.WUint8(Time)
		b.WInt64(v.UnixNano())
	case []time.Time:
		b.WUint8(Times)
		b.WUint32(uint32(len(v)))
		for _, t := range v {
			b.WInt64(t.UnixMicro())
		}
	case map[string]int32:
		b.WUint8(MInt32)
		b.WMInt32(v)
	case map[string]int64:
		b.WUint8(MInt64)
		b.WMInt64(v)
	case map[uint16]uint16:
		b.WUint8(Uint16MUint16)
		b.WUint16MUint16(v)
	case map[int32]int32:
		b.WUint8(Int32MInt32)
		b.WInt32MInt32(v)
	case map[int32]int64:
		b.WUint8(Int32MInt64)
		b.WInt32MInt64(v)
	case map[int64]int32:
		b.WUint8(Int64MInt32)
		b.WInt64MInt32(v)
	case map[int64]int64:
		b.WUint8(Int64MInt64)
		b.WInt64MInt64(v)
	case MS:
		b.WUint8(MStr)
		b.WMStr(v)
	case M:
		b.WUint8(MAny)
		return b.WMAny(v)
	case map[any]any:
		b.WUint8(AnyMAny)
		return b.WAnyMAny(v)
	default:
		m := make(M)
		e := mapstructure.Decode(v, &m)
		if e != nil {
			return NewErr(EcNotImplement, M{
				"error": e.Error(),
			})
		}
		b.WUint8(MAny)
		return b.WMAny(m)
	}
	return nil
}
func (b *ByteBuffer) RAny() (v any, err *Err) {
	t, err := b.RUint8()
	if err != nil {
		return nil, err
	}
	switch t {
	case Bool:
		return b.RBool()
	case Bools:
		return b.RBools()
	case Uint8:
		return b.RUint8()
	case Bytes:
		return b.RBytes()
	case Int8:
		return b.RInt8()
	case Int8s:
		return b.RInt8s()
	case Uint16:
		return b.RUint16()
	case Uint16s:
		return b.RUint16s()
	case Int16:
		return b.RInt16()
	case Int16s:
		return b.RInt16s()
	case Uint32:
		return b.RUint32()
	case Uint32s:
		return b.RUint32s()
	case Int:
		return b.RInt()
	case Ints:
		return b.RInts()
	case Int32:
		return b.RInt32()
	case Int32s:
		return b.RInt32s()
	case Uint64:
		return b.RUint64()
	case Uint64s:
		return b.RUint64s()
	case Int64:
		return b.RInt64()
	case Int64s:
		return b.RInt64s()
	case Float32:
		return b.RFloat32()
	case Float32s:
		return b.RFloat32s()
	case Float64:
		return b.RFloat64()
	case Float64s:
		return b.RFloat64s()
	case Vector2:
		return b.RVec2()
	case Vector2s:
		return b.RVec2s()
	case Vector3:
		return b.RVec3()
	case Vector3s:
		return b.RVec3s()
	case Str:
		return b.RString()
	case Strs:
		return b.RStrings()
	case Time:
		v, e := b.RInt64()
		if e != nil {
			return time.Time{}, e
		}
		return time.UnixMicro(v), nil
	case Times:
		c, e := b.RUint32()
		if e != nil {
			return nil, e
		}
		ts := make([]time.Time, c)
		for i := uint32(0); i < c; i++ {
			t, e := b.RInt64()
			if e != nil {
				return nil, e
			}
			ts[i] = time.UnixMicro(t)
		}
		return ts, nil
	case MInt32:
		return b.RMInt32()
	case MInt64:
		return b.RMInt64()
	case MStr:
		return b.RMStr()
	case MAny:
		m := make(M)
		err := b.RMAny(m)
		if err != nil {
			return nil, err
		}
		return m, nil
	case Uint16MUint16:
		return b.RUint16MUint16()
	case Int32MInt32:
		return b.RInt32MInt32()
	case Int32MInt64:
		return b.RInt32MInt64()
	case Int64MInt32:
		return b.RInt64MInt32()
	case Int64MInt64:
		return b.RInt64MInt64()
	case AnyMAny:
		m := make(map[any]any)
		err := b.RAnyMAny(m)
		if err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, NewErr(EcNotImplement, M{
			"type": t,
		})
	}
}

func (b *ByteBuffer) WAny2(m map[any]any) (err *Err) {
	b.WUint16(uint16(len(m)))
	for k, v := range m {
		err = b.WAny(k)
		if err != nil {
			return
		}
		err = b.WAny(v)
		if err != nil {
			return
		}
	}
	return
}

func (b *ByteBuffer) WErr(err *Err) *Err {
	b.WUint16(err.Code())
	bytes, err := JsonMarshal(err.Params())
	if err != nil {
		return err
	}
	b.WBytes(bytes)
	return nil
}

func (b *ByteBuffer) RErr() (e *Err, re *Err) {
	code, err := b.RUint16()
	if err != nil {
		return nil, err
	}
	bytes := b.RAvailable()
	params := M{}
	err = JsonUnmarshal(bytes, params)
	if err != nil {
		return nil, err
	}
	return NewNoStackErr(code, params), nil
}

func (b *ByteBuffer) WriteTo(writer io.Writer) (n int, err error) {
	return writer.Write(b.bytes[:b.len])
}

// Dispose 释放后不要再使用数据
func (b *ByteBuffer) Dispose() {
	if !b.canRecycle {
		return
	}
	b.canRecycle = false
	RecycleBytes(b.bytes)
}

func init() {
	gob.Register(M{})
	gob.Register([]any{})
	gob.Register(Vec2{})
	gob.Register(Vec3{})
}

func PbMarshal(pkt IMsg) ([]byte, error) {
	return proto.Marshal(pkt)
}

func PbUnmarshal(data []byte, pkt IMsg) error {
	return proto.Unmarshal(data, pkt)
}
