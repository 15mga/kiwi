package util

type (
	TErrCode = uint16
)

type (
	Fn                   func()
	ToBool               func() bool
	ToInt64              func() int64
	ToStr                func() string
	ToStrBool            func() (string, bool)
	ToAny                func() any
	ToM                  func() M
	ToErr                func() *Err
	ToErrCode            func() TErrCode
	ToFnInt64Fn          func() FnStrFn
	ToMsg                func() IMsg
	FnBool               func(bool)
	BoolToErr            func(bool) *Err
	FnBoolErr            func(bool, *Err)
	FnUint16             func(uint16)
	FnUint16Bool         func(uint16, bool)
	FnUint16Bytes        func(uint16, []byte)
	FnUint16Err          func(uint16, *Err)
	FnUint162Int642Bytes func(uint16, uint16, int64, int64, []byte)
	FnUint16Int642Bytes  func(uint16, int64, int64, []byte)
	Uint16ToBool         func(uint16) bool
	Uint16Int64ToUint16  func(uint16, int64) uint16
	FnInt                func(int)
	FnAnySlc             func([]any)
	FnAnySlc2            func(...any)
	FnIntAnySlc          func(int, []any)
	FnIntAnySlc2         func(int, []any, []any)
	FnInt32              func(int32)
	FnUint32             func(uint32)
	FnErr                func(*Err)
	FnMsg                func(IMsg)
	FnInt64              func(int64)
	FnInt64Bool          func(int64, bool)
	FnInt642             func(int64, int64)
	FnInt64Str           func(int64, string)
	FnInt64Bytes         func(int64, []byte)
	FnInt64M             func(int64, M)
	FnInt64MBytes        func(int64, M, []byte)
	FnInt64Any           func(int64, any)
	FnInt64Err           func(int64, *Err)
	FnInt64MErr          func(int64, M, *Err)
	FnInt643             func(int64, int64, int64)
	FnInt642Bool         func(int64, int64, bool)
	FnInt642Any          func(int64, int64, any)
	FnInt642Err          func(int64, int64, *Err)
	FnInt64StrBool       func(int64, string, bool)
	FnInt64MUint16       func(int64, M, uint16)
	FnInt64MMsg          func(int64, M, IMsg)
	Int64ToInt64         func(int64) int64
	Int64ToStr           func(int64) string
	Int64ToBool          func(int64) bool
	Int64ToErrCode       func(int64) TErrCode
	Int64ToStrErr        func(int64) (string, *Err)
	Int64ToInt64Err      func(int64) (int64, *Err)
	Int64AnyToBool       func(int64, any) bool
	FnStr                func(string)
	StrToBool            func(string) bool
	StrToStr             func(string) string
	FnStrBool            func(string, bool)
	StrIntToBool         func(string, int) *Err
	StrInt64ToBool       func(string, int64) *Err
	FnStrBytes           func(string, []byte)
	FnStrAny             func(string, any)
	FnStrFn              func(string, Fn)
	FnStrErr             func(string, *Err)
	FnStr2Bool           func(string, string, bool)
	StrFnToErr           func(string, Fn) *Err
	StrToStr2Err         func(string) (string, string, *Err)
	StrToBytesErr        func(string) ([]byte, *Err)
	StrAnyToErr          func(string, any) *Err
	StrBytesToBool       func(string, []byte) bool
	Str2Int64ToBoolBytes func(string, string, int64) (bool, *Err)
	Str2BytesToBytesErr  func(string, string, []byte) ([]byte, *Err)
	Str162ErrToBytesErr  func(string, string, *Err) ([]byte, *Err)
	FnStrSlc             func([]string)
	FnBytes              func([]byte)
	FnBytesSlc           func([][]byte)
	BytesToUint16        func([]byte) uint16
	BytesToUint32        func([]byte) uint32
	BytesToBytes         func([]byte) []byte
	BytesToM             func([]byte) M
	BytesToErr           func([]byte) *Err
	BytesToInt642Err     func([]byte) (int64, int64, *Err)
	BytesToAnyErr        func([]byte) (any, *Err)
	BytesAnyToError      func([]byte, any) error
	BytesAnyToErr        func([]byte, any) *Err
	FnAny                func(any)
	FnAnyErr             func(any, *Err)
	AnyToBool            func(any) bool
	AnyToInt64           func(any) int64
	AnyToAny             func(any) any
	AnyToErr             func(any) *Err
	AnyToAnyBool         func(any) (any, bool)
	AnyToBytesError      func(any) ([]byte, error)
	AnyBoolToAnyBool     func(any, bool) (any, bool)
	AnyBoolToBool        func(any, bool) bool
	AnyErrToBool         func(any, *Err) bool
	FnM                  func(M)
	FnMBool              func(M, bool)
	FnMAny               func(M, any)
	FnM2Bool             func(M, M, bool)
	MToBool              func(M) bool
	MToInt64             func(M) int64
	MToAny               func(M) any
	MToBytes             func(M) []byte
	MToErr               func(M) *Err
	FnMErr               func(M, *Err)
	FnMUint32            func(M, uint32)
	Compare[T any]       func(v1, v2 T) int
	FnMapBool            func(map[string]bool)
)

func Default[T any]() (v T) {
	return
}

func (f Fn) Invoke() {
	if f == nil {
		return
	}
	f()
}

func (f FnInt642) Invoke(v1, v2 int64) {
	if f == nil {
		return
	}
	f(v1, v2)
}

func (f FnInt64Any) Invoke(v int64, obj any) {
	if f == nil {
		return
	}
	f(v, obj)
}

func (f FnInt64MMsg) Invoke(v int64, head M, obj IMsg) {
	if f == nil {
		return
	}
	f(v, head, obj)
}

func (f FnErr) Invoke(err *Err) {
	if f == nil {
		return
	}
	f(err)
}

func (f FnInt64) Invoke(v int64) {
	if f == nil {
		return
	}
	f(v)
}

func (f FnInt64Str) Invoke(v int64, str string) {
	if f == nil {
		return
	}
	f(v, str)
}

func (f FnInt64Err) Invoke(v int64, err *Err) {
	if f == nil {
		return
	}
	f(v, err)
}

func (f FnInt64MErr) Invoke(v int64, head M, err *Err) {
	if f == nil {
		return
	}
	f(v, head, err)
}

func (f FnStr) Invoke(str string) {
	if f == nil {
		return
	}
	f(str)
}

func (m M) IndentJson() []byte {
	bytes, err := JsonMarshalIndent(m, "", " ")
	if err != nil {
		panic(err)
	}
	return bytes
}

type Id interface {
	int32 | uint32 | int64 | uint64 | string
}
