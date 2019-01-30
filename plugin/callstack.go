package plugin

type Callstack interface {
	Push(entry string)
	Pop(entry string)
	Has(entry string) bool
	Free()
	List() []string
}

func NewCallstack() Callstack {
	return &callstack{stack: make(map[string]int), index: 0}
}

type callstack struct {
	stack  map[string]int
	sorted []string
	index  int
}

func (c *callstack) Free() {
	c.stack = make(map[string]int)
	c.index = 0
}

func (c *callstack) Pop(entry string) {
	delete(c.stack, entry)
	c.index--
}

func (c *callstack) Push(entry string) {
	c.stack[entry] = c.index
	c.index++
}

func (c *callstack) List() []string {
	res := make([]string, len(c.stack))
	for s, si := range c.stack {
		res[si] = s
	}
	return res
}

func (c *callstack) Has(entry string) bool {
	_, ok := c.stack[entry]
	return ok
}
