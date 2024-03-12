package ecs

import (
	"sync"
)

func newBuffer() *Buffer {
	b := &Buffer{}
	return b
}

type Buffer struct {
	head *job
	tail *job
}

func (b *Buffer) push(cmd JobName, data ...any) {
	j := _JobPool.Get().(*job)
	j.Name = cmd
	j.Data = data

	if b.head != nil {
		b.tail.next = j
	} else {
		b.head = j
	}
	b.tail = j
}

var (
	_JobPool = sync.Pool{
		New: func() any {
			return &job{}
		},
	}
)

type job struct {
	Name JobName
	Data []any
	next *job
}

type (
	JobName     = string
	FnBufferJob func(JobName, *Buffer, []any)
	FnJob       func([]any)
)
