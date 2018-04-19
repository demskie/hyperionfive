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
	objMap map[uint64]*customSpatial
}

type customSpatial struct {
	rct *rtreego.Rect
	id  uint64
}

func (sp *customSpatial) Bounds() *rtreego.Rect {
	return sp.rct
}

// NewTree ...
func NewTree() *Tree {
	return &Tree{
		rt:     rtreego.NewTree(2, 10, 10),
		mtx:    &sync.RWMutex{},
		objMap: make(map[uint64]*customSpatial),
	}
}

// Insert ...
func (t *Tree) Insert(id uint64, x, y float64) error {
	rect, err := rtreego.NewRect(rtreego.Point{x, y}, []float64{0.01, 0.01})
	if err != nil {
		return err
	}
	t.mtx.Lock()
	_, exists := t.objMap[id]
	if exists {
		t.mtx.Unlock()
		return errors.New("duplicate n value")
	}
	t.objMap[id] = &customSpatial{rect, id}
	t.rt.Insert(&customSpatial{rct: rect})
	t.mtx.Unlock()
	return nil
}

// Delete ...
func (t *Tree) Delete(id uint64) error {
	t.mtx.Lock()
	obj, exists := t.objMap[id]
	if !exists {
		t.mtx.Unlock()
		return errors.New("id does not exist")
	}
	delete(t.objMap, id)
	t.rt.Delete(obj)
	t.mtx.Unlock()
	return nil
}

// Object ...
type Object struct {
	ID   uint64
	X, Y float64
}

// NearestNeighbors ...
func (t *Tree) NearestNeighbors(id uint64, number int) (results []Object, err error) {
	t.mtx.RLock()
	obj, exists := t.objMap[id]
	if !exists {
		t.mtx.RUnlock()
		return []Object{}, errors.New("n value does not exist")
	}
	spatials := t.rt.NearestNeighbors(1+number, rtreego.Point{
		obj.rct.PointCoord(0),
		obj.rct.PointCoord(1),
	})
	for i := 0; i < len(spatials); i++ {
		if spatials[i].(*customSpatial).id == id {
			spatials = append(spatials[:i], spatials[i+1:]...)
			break
		}
	}
	results = make([]Object, number)
	for i, spatial := range spatials[:number] {
		if spatial.(*customSpatial).id != id {
			results[i] = Object{
				id,
				obj.rct.PointCoord(0),
				obj.rct.PointCoord(1),
			}
			break
		}
	}
	t.mtx.RUnlock()
	return results, nil
}
