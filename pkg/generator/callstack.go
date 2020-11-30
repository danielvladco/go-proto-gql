package generator

type Callstack interface {
	Push(entry interface{})
	Pop(entry interface{})
	Has(entry interface{}) bool
	Free()
	Len() int
}

func NewCallstack() Callstack {
	return &callstack{stack: make(map[interface{}]int), index: 0}
}

type callstack struct {
	stack  map[interface{}]int
	sorted []string
	index  int
}

func (c *callstack) Free() {
	c.stack = make(map[interface{}]int)
	c.index = 0
}

func (c *callstack) Pop(entry interface{}) {
	delete(c.stack, entry)
	c.index--
}

func (c *callstack) Push(entry interface{}) {
	c.stack[entry] = c.index
	c.index++
}

func (c *callstack) List() []interface{} {
	res := make([]interface{}, len(c.stack))
	for s, si := range c.stack {
		res[si] = s
	}
	return res
}

func (c *callstack) Has(entry interface{}) bool {
	_, ok := c.stack[entry]
	return ok
}

func (c *callstack) Len() int {
	return len(c.stack)
}
