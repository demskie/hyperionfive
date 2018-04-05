package simpletree

import (
	"errors"
	"sync"

	"github.com/dhconnelly/rtreego"
)

// Tree ..
type Tree struct {
	rt     *rtreego.Rtree
	mtx    *sync.RWMutex
	objMap map[string]*customSpatial
}

type customSpatial struct {
	rct *rtreego.Rect
	nm  string
}

func (sp *customSpatial) Bounds() *rtreego.Rect {
	return sp.rct
}

// NewTree ...
func NewTree() *Tree {
	return &Tree{
		rt:     rtreego.NewTree(2, 5, 10),
		mtx:    &sync.RWMutex{},
		objMap: make(map[string]*customSpatial),
	}
}

// Insert ...
func (t *Tree) Insert(name string, x, y float64) error {
	rect, err := rtreego.NewRect(rtreego.Point{x, y}, []float64{0.01, 0.01})
	if err != nil {
		return err
	}
	t.mtx.Lock()
	_, exists := t.objMap[name]
	if exists {
		t.mtx.Unlock()
		return errors.New("duplicate n value")
	}
	t.objMap[name] = &customSpatial{rect, name}
	t.rt.Insert(&customSpatial{rct: rect})
	t.mtx.Unlock()
	return nil
}

// Delete ...
func (t *Tree) Delete(n string) error {
	t.mtx.Lock()
	obj, exists := t.objMap[n]
	if !exists {
		t.mtx.Unlock()
		return errors.New("n value does not exist")
	}
	delete(t.objMap, n)
	t.rt.Delete(obj)
	t.mtx.Unlock()
	return nil
}

// Object ...
type Object struct {
	Name string
	X, Y float64
}

// NearestNeighbors ...
func (t *Tree) NearestNeighbors(name string, number int) (results []Object, err error) {
	t.mtx.RLock()
	obj, exists := t.objMap[name]
	if !exists {
		t.mtx.RUnlock()
		return []Object{}, errors.New("n value does not exist")
	}
	spatials := t.rt.NearestNeighbors(1+number, rtreego.Point{
		obj.rct.PointCoord(0),
		obj.rct.PointCoord(1),
	})
	for i := 0; i < len(spatials); i++ {
		if spatials[i].(*customSpatial).nm == name {
			spatials = append(spatials[:i], spatials[i+1:]...)
			break
		}
	}
	results = make([]Object, number)
	for i, spatial := range spatials[:number] {
		if spatial.(*customSpatial).nm != name {
			results[i] = Object{
				name,
				obj.rct.PointCoord(0),
				obj.rct.PointCoord(1),
			}
			break
		}
	}
	t.mtx.RUnlock()
	return results, nil
}
