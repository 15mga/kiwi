package util

var (
	_Command *Command[M]
)

func DefaultCmd(m M) {
	_Command = NewCommand[M](m)
}

func Cmd() *Command[M] {
	return _Command
}

func NewCommand[T any](data T) *Command[T] {
	return &Command[T]{
		data:          data,
		nameToHandler: make(map[string]CmdHandler[T]),
	}
}

type CmdHandler[T any] func(T, any) *Err

type Command[T any] struct {
	data          T
	nameToHandler map[string]CmdHandler[T]
}

func (c *Command[T]) Data() T {
	return c.data
}

func (c *Command[T]) Bind(name string, handler CmdHandler[T]) {
	c.nameToHandler[name] = handler
}

func (c *Command[T]) Unbind(name string) bool {
	if _, ok := c.nameToHandler[name]; !ok {
		return false
	}
	delete(c.nameToHandler, name)
	return true
}

func (c *Command[T]) Process(name string, data any) *Err {
	handler, ok := c.nameToHandler[name]
	if !ok {
		return NewErr(EcNotExist, M{
			"name": name,
		})
	}
	return handler(c.data, data)
}
