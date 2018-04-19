package randutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/rand"
	"time"

	cryptorand "crypto/rand"
)

// WeightedObj ...
type WeightedObj struct {
	Weight int
	Item   interface{}
}

// WeightedChoice ...
func WeightedChoice(choices []WeightedObj, rnum *rand.Rand) (WeightedObj, error) {
	max := 0
	for _, choice := range choices {
		max += choice.Weight
	}
	num := rnum.Intn(max)
	for _, choice := range choices {
		num -= choice.Weight
		if num <= 0 {
			return choice, nil
		}
	}
	err := errors.New("weightedChoice func is broken")
	return WeightedObj{}, err
}

// GetUniqueMathRnum ...
func GetUniqueMathRnum() *rand.Rand {
	var randomness int64
	b := make([]byte, 8)
	cryptorand.Read(b)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &randomness)
	return rand.New(rand.NewSource(randomness))
}

// GetMultipleUniqueMathRnum ...
func GetMultipleUniqueMathRnum(numCores int) []*rand.Rand {
	result := make([]*rand.Rand, numCores)
	for i := 0; i < numCores; i++ {
		result[i] = GetUniqueMathRnum()
	}
	return result
}

// GetMathRnum ...
func GetMathRnum() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GetPosOrNegOne ...
func GetPosOrNegOne(rnum *rand.Rand) int {
	if rnum.Intn(2) == 0 {
		return 1
	}
	return -1
}
