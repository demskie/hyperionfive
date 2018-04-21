package simpletree

import "testing"

func TestTreeLookup(t *testing.T) {
	tree := NewTree()

	err := tree.Insert(0, 5, 5)
	if err != nil {
		t.Error(err)
	}

	err = tree.Insert(1, 6, 6)
	if err != nil {
		t.Error(err)
	}

	neighbors, err := tree.NearestNeighbors(0, 99)
	if err != nil {
		t.Error(err)
	}
	if len(neighbors) != 1 {
		t.Errorf("returned %v neighbors", len(neighbors))
	}
	if neighbors[0].ID != 1 || neighbors[0].X != 6 || neighbors[0].Y != 6 {
		t.Error("NearestNeighbor function is broken")
	}
}
