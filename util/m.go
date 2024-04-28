package util

type (
	M    map[string]any
	MS   map[string]string
	M32  map[string]int32
	M64  map[string]int64
	MF32 map[string]float32
	MF64 map[string]float64
)

func MGet[T any](m M, key string) (T, bool) {
	v, ok := m[key]
	if !ok {
		return Default[T](), false
	}
	v1, ok := v.(T)
	return v1, ok
}

func MGet2[T any](m M, keys ...string) (T, bool) {
	switch len(keys) {
	case 0:
		return Default[T](), false
	case 1:
		return MGet[T](m, keys[0])
	default:
		sm, ok := MGet[M](m, keys[0])
		if !ok {
			return Default[T](), false
		}
		return MGet2[T](sm, keys[1:]...)
	}
}

func MGetOrSet[T any](m M, key string, new func() T) T {
	v, ok := m[key]
	if !ok {
		v = new()
		m[key] = v
	}
	return v.(T)
}

func MPop[T any](m M, key string) (T, bool) {
	v, ok := m[key]
	if !ok {
		return Default[T](), true
	}
	delete(m, key)
	v1, ok := v.(T)
	return v1, ok
}

func MReplace[T any](m M, key string, val T) (T, bool) {
	oldVal, ok := MGet[T](m, key)
	if !ok {
		return Default[T](), false
	}
	m[key] = val
	return oldVal, true
}

func MUpdate[T any](m M, key string, fn func(T) T) bool {
	oldVal, ok := MGet[T](m, key)
	m[key] = fn(oldVal)
	return ok
}

func NewM(bytes []byte, m M) *Err {
	var buff ByteBuffer
	buff.InitBytes(bytes)
	return buff.RMAny(m)
}

func (m M) Replace(key string, val any) (oldVal any, ok bool) {
	oldVal, ok = m[key]
	m[key] = val
	return
}

func (m M) Update(key string, action AnyToAny) (ok bool) {
	oldVal, ok := m[key]
	m[key] = action(oldVal)
	return
}

func (m M) Set(key string, val any) {
	m[key] = val
}

func (m M) Set2(val any, keys ...string) *Err {
	switch len(keys) {
	case 0:
		return NewErr(EcParamsErr, M{
			"error": "keys empty",
		})
	case 1:
		m[keys[0]] = val
		return nil
	default:
		k := keys[0]
		o, ok := m[k]
		if !ok {
			o = M{}
			m[k] = o
		}
		sm, ok := o.(M)
		if !ok {
			return NewErr(EcWrongType, M{
				"error": "not M",
				"key":   k,
				"value": o,
			})
		}
		return sm.Set2(val, keys[1:]...)
	}
}

func (m M) Del(key string) {
	delete(m, key)
}

func (m M) Get(key string) (any, bool) {
	val, ok := m[key]
	if !ok {
		return nil, false
	}
	return val, true
}

func (m M) MustGet(key string, def any) any {
	val, ok := m[key]
	if !ok {
		m[key] = def
		return def
	}
	return val
}

func (m M) Has(key string) (ok bool) {
	_, ok = m[key]
	return
}

func (m M) ToJson() ([]byte, *Err) {
	bytes, err := JsonMarshal(m)
	return bytes, WrapErr(EcMarshallErr, err)
}

func (m M) FromJson(bytes []byte) *Err {
	return JsonUnmarshal(bytes, &m)
}

func (m M) ToGob() ([]byte, *Err) {
	bytes, err := GobMarshal(m)
	return bytes, WrapErr(EcMarshallErr, err)
}

func (m M) FromGob(bytes []byte) *Err {
	e := GobUnmarshal(bytes, &m)
	if e != nil {
		return WrapErr(EcUnmarshallErr, e)
	}
	return nil
}

func (m M) ToBytes() ([]byte, *Err) {
	var buff ByteBuffer
	buff.InitCap(64)
	err := buff.WMAny(m)
	if err != nil {
		return nil, err
	}
	return buff.All(), nil
}

func (m M) FromBytes(bytes []byte) *Err {
	var buff ByteBuffer
	buff.InitBytes(bytes)
	return buff.RMAny(m)
}

func (m M) ToSlice(slc *[]any) {
	for k, v := range m {
		*slc = append(*slc, k, v)
	}
}

func (m M) Copy() M {
	n := make(M, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

func (m M) CopyTo(n M) {
	for k, v := range m {
		n[k] = v
	}
}
