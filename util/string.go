package util

import (
	"encoding/hex"
	"strconv"
	"strings"
	"unsafe"
)

func SplitUint16(sep, str string) ([]uint16, *Err) {
	items := strings.Split(str, sep)
	vals := make([]uint16, len(items))
	for i, item := range items {
		v, e := strconv.Atoi(item)
		if e != nil {
			return nil, WrapErr(EcParseErr, e)
		}
		vals[i] = uint16(v)
	}
	return vals, nil
}

func SplitInt64(sep, str string) ([]int64, *Err) {
	items := strings.Split(str, sep)
	vals := make([]int64, len(items))
	for i, item := range items {
		v, e := strconv.ParseInt(item, 10, 64)
		if e != nil {
			return nil, WrapErr(EcParseErr, e)
		}
		vals[i] = v
	}
	return vals, nil
}

func BytesToStr(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StrToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

const (
	_UpToLowerOffset = byte('a' - 'A')
)

func ToBigHump(str string) string {
	bytes := []byte(str)
	var sb strings.Builder
	l := len(bytes)
	sb.Grow(l)
	h := false
	first := true
	for i := 0; i < l; i++ {
		s := bytes[i]
		if s == '_' {
			h = true
			continue
		}
		if s >= 'a' && s <= 'z' && (h || first) {
			h = false
			first = false
			sb.WriteByte(s - _UpToLowerOffset)
			continue
		}
		sb.WriteByte(s)
	}
	return sb.String()
}

func ToUnderline(str string) string {
	bytes := []byte(str)
	var sb strings.Builder
	sb.Grow(len(bytes))
	for i, s := range bytes {
		if s < 'A' || s > 'Z' {
			sb.WriteByte(s)
		} else {
			if i > 0 {
				sb.WriteByte('_')
			}
			sb.WriteByte(s + _UpToLowerOffset)
		}
	}
	return sb.String()
}

func Hex(bytes []byte) string {
	str := hex.EncodeToString(bytes)
	l := len(str)
	s := ""
	i := 0
	for i < l {
		s += str[i : i+2]
		s += " "
		i += 2
	}
	return s
}

func SplitWords(str string, words *[]string) {
	var (
		start = 0
		end   = 0
	)
	bytes := []byte(str)
	l := len(bytes)
	for ; end < l; end++ {
		s := bytes[end]
		if end == 0 || s < 'A' || s > 'Z' {
			continue
		}

		*words = append(*words, BytesToStr(bytes[start:end]))
		start = end
	}
	if start < l {
		*words = append(*words, BytesToStr(bytes[start:l]))
	}
}

func ParseInt[T Num](s string) (v T, err *Err) {
	i, e := strconv.ParseInt(s, 10, 64)
	if e != nil {
		return 0, NewErr(EcParseErr, M{
			"string": s,
		})
	}
	v = T(i)
	return
}

func StringsJoin(sep string, slc ...string) string {
	return strings.Join(slc, sep)
}
