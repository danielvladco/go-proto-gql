package generator

import "github.com/jhump/protoreflect/desc"

type Callstack interface {
	Push(entry desc.Descriptor)
	Pop(entry desc.Descriptor)
	Has(entry desc.Descriptor) bool
	Free()
	List() []desc.Descriptor
	Len() int
}

func NewCallstack() Callstack {
	return &callstack{stack: make(map[desc.Descriptor]int), index: 0}
}

type callstack struct {
	stack  map[desc.Descriptor]int
	sorted []string
	index  int
}

func (c *callstack) Free() {
	c.stack = make(map[desc.Descriptor]int)
	c.index = 0
}

func (c *callstack) Pop(entry desc.Descriptor) {
	delete(c.stack, entry)
	c.index--
}

func (c *callstack) Push(entry desc.Descriptor) {
	c.stack[entry] = c.index
	c.index++
}

func (c *callstack) List() []desc.Descriptor {
	res := make([]desc.Descriptor, len(c.stack))
	for s, si := range c.stack {
		res[si] = s
	}
	return res
}

func (c *callstack) Has(entry desc.Descriptor) bool {
	_, ok := c.stack[entry]
	return ok
}

func (c *callstack) Len() int {
	return len(c.stack)
}
