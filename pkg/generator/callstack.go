package generator

type Callstack interface {
	Push(entry interface{})
	Pop(entry interface{})
	Has(entry interface{}) bool
}

func NewCallstack() Callstack {
	return &callstack{stack: make(map[interface{}]int), index: 0}
}

type callstack struct {
	stack  map[interface{}]int
	sorted []string
	index  int
}

func (c *callstack) Pop(entry interface{}) {
	delete(c.stack, entry)
	c.index--
}

func (c *callstack) Push(entry interface{}) {
	c.stack[entry] = c.index
	c.index++
}

func (c *callstack) Has(entry interface{}) bool {
	_, ok := c.stack[entry]
	return ok
}
