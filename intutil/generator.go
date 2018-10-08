package intutil

import (
	"math/rand"
	"sync"

	"github.com/demskie/randutil"
)

// Int32Generator ...
type Int32Generator struct {
	mtx  *sync.Mutex
	rnum *rand.Rand
	used map[int32]struct{}
}

// NewInt32Generator ...
func NewInt32Generator() *Int32Generator {
	return &Int32Generator{
		mtx:  &sync.Mutex{},
		rnum: randutil.CreateUniqueMathRnum(),
		used: map[int32]struct{}{},
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
	for {
		val = g.rnum.Int31()
		_, exists := g.used[val]
		if !exists {
			break
		}
	}
	g.mtx.Unlock()
	return val
}

// Remove deletes this integer
func (g *Int32Generator) Remove(i int32) {
	g.mtx.Lock()
	delete(g.used, i)
	g.mtx.Unlock()
}
