package stringdatastore

import "sync"

// References hold the data necessary for lookups
type References struct {
	mtx           *sync.RWMutex
	stringToUint  map[string]uint64
	uintToString  map[uint64]string
	unusedNumbers map[uint64]struct{}
}

// New returns an initialized References object
func New() *References {
	return &References{
		mtx:           &sync.RWMutex{},
		stringToUint:  map[string]uint64{},
		uintToString:  map[uint64]string{},
		unusedNumbers: map[uint64]struct{}{},
	}
}

// Len returns the number of strings being stored
func (r *References) Len() int {
	r.mtx.RLock()
	val := len(r.stringToUint)
	r.mtx.RUnlock()
	return val
}

// CreateReferenceUint returns a unique integer to represent the string
func (r *References) CreateReferenceUint(s string) uint64 {
	r.mtx.RLock()
	i, exists := r.stringToUint[s]
	r.mtx.RUnlock()
	if !exists {
		r.mtx.Lock()
		if len(r.unusedNumbers) == 0 {
			i = uint64(len(r.stringToUint))
		} else {
			for k := range r.unusedNumbers {
				i = k
				break
			}
			delete(r.unusedNumbers, i)
		}
		r.stringToUint[s] = i
		r.uintToString[i] = s
		r.mtx.Unlock()
	}
	return i
}

// GetString will return the string represented by the integer if there is one
func (r *References) GetString(i uint64) string {
	r.mtx.RLock()
	val, exists := r.uintToString[i]
	r.mtx.RUnlock()
	if !exists {
		return ""
	}
	return val
}

// DeleteString removes any references for this string if it exists
func (r *References) DeleteString(s string) {
	r.mtx.RLock()
	i, exists := r.stringToUint[s]
	r.mtx.RUnlock()
	if exists {
		r.mtx.Lock()
		delete(r.stringToUint, s)
		delete(r.uintToString, i)
		r.unusedNumbers[i] = struct{}{}
		r.mtx.Unlock()
	}
}
