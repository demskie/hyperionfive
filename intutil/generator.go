package intutil

import "sync"

// Int32Generator ...
type Int32Generator struct {
	mtx    *sync.Mutex
	used   map[int32]struct{}
	unused map[int32]struct{}
}

// NewInt32Generator ...
func NewInt32Generator() *Int32Generator {
	return &Int32Generator{
		mtx:    &sync.Mutex{},
		used:   map[int32]struct{}{},
		unused: map[int32]struct{}{},
	}
}

// Len returns the number of integers being used
func (g *Int32Generator) Len() int {
	g.mtx.Lock()
	val := len(g.used)
	g.mtx.Unlock()
	return val
}

// Create ...
func (g *Int32Generator) Create() int32 {
	var val int32
	g.mtx.Lock()
	if len(g.unused) > 0 {
		for k := range g.unused {
			val = k
			break
		}
		delete(g.unused, val)
	} else {
		val = int32(len(g.used))
		g.used[val] = struct{}{}
	}
	g.mtx.Unlock()
	return val
}

// Remove deletes this integer
func (g *Int32Generator) Remove(i int32) {
	g.mtx.Lock()
	_, exists := g.used[i]
	if exists {
		delete(g.used, i)
		g.unused[i] = struct{}{}
	}
	g.mtx.Unlock()
}
