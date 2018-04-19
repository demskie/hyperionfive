package stringdatastore

import (
	"math"
	"sync"
)

// References hold the data necessary for lookups
type References struct {
	mtx         *sync.RWMutex
	stringToInt map[string]int32
	intToString map[int32]string
	emptyKeys   []int32
}

// New returns an initialized References object
func New() *References {
	return &References{
		mtx:         &sync.RWMutex{},
		stringToInt: map[string]int32{},
		intToString: map[int32]string{},
		emptyKeys:   []int32{},
	}
}

// Len returns the number of strings being stored
func (r *References) Len() int {
	r.mtx.RLock()
	val := len(r.stringToInt)
	r.mtx.RUnlock()
	return val
}

// GetReferenceInt returns a unique integer to represent the string
func (r *References) GetReferenceInt(s string) int32 {
	r.mtx.RLock()
	i, exists := r.stringToInt[s]
	r.mtx.RUnlock()
	if !exists {
		r.mtx.Lock()
		if len(r.emptyKeys) != 0 {
			i = r.emptyKeys[len(r.emptyKeys)]
			r.emptyKeys = r.emptyKeys[:len(r.emptyKeys)-1]
		} else {
			i = int32(len(r.stringToInt))
		}
		r.stringToInt[s] = i
		r.intToString[i] = s
		r.mtx.Unlock()
	}
	if i >= math.MaxInt32 {
		panic("overflow detected")
	}
	return i
}

// GetString will return the string represented by the integer if there is one
func (r *References) GetString(i int32) string {
	r.mtx.RLock()
	val, exists := r.intToString[i]
	r.mtx.RUnlock()
	if !exists {
		return ""
	}
	return val
}

// DeleteString removes any references for this string if it exists
func (r *References) DeleteString(s string) {
	r.mtx.RLock()
	i, exists := r.stringToInt[s]
	r.mtx.RUnlock()
	if exists {
		r.mtx.Lock()
		delete(r.stringToInt, s)
		delete(r.intToString, i)
		if len(r.intToString) == 0 {
			r.emptyKeys = []int32{}
		} else {
			r.emptyKeys = append(r.emptyKeys, i)
		}
		r.mtx.Unlock()
	}
}
