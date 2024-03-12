package worker

import "testing"

func TestFnv(t *testing.T) {
	v := FnvInt64(1000) & 16
	t.Log(v)
	v = FnvStr("scene") & 16
	t.Log(v)
}
