package ds

func NewIdLink[KT comparable, VT any]() *IdLink[KT, VT] {
	return &IdLink[KT, VT]{}
}

type IdLink[KT comparable, VT any] struct {
	id KT
	Link[VT]
}

func (q *IdLink[KT, VT]) Id() KT {
	return q.id
}

func (q *IdLink[KT, VT]) SetId(id KT) {
	q.id = id
}

func (q *IdLink[KT, VT]) Push(val VT) {
	q.Link.Push(val)
}

func (q *IdLink[KT, VT]) PopAll() *LinkElem[VT] {
	return q.Link.PopAll()
}
