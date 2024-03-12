package ecs

func NewComponent(t TComponent) Component {
	c := Component{
		typ: t,
	}
	return c
}

type Component struct {
	typ    TComponent
	entity *Entity
}

func (c *Component) Type() TComponent {
	return c.typ
}

func (c *Component) Init() {

}

func (c *Component) Start() {
}

func (c *Component) Dispose() {
}

func (c *Component) Entity() *Entity {
	return c.entity
}

func (c *Component) setEntity(entity *Entity) {
	c.entity = entity
}
