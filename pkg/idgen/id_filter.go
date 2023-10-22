package idgen

import (
	"fmt"
	"math/rand"
)

type IdFilter func(id int64) (int64, error)

func AddRandomFilter(bit uint) func(id int64) (int64, error) {
	checkOverFlow := func(x int64, shift uint) bool {
		if shift >= 64 {
			return true
		}
		result := x << shift
		return result>>shift != x
	}

	return func(id int64) (int64, error) {
		if checkOverFlow(id, bit) {
			return id, fmt.Errorf("AddRandomFilter %d overflow", bit)
		}
		randId := rand.Intn(1 << bit)
		return ((id << bit) + int64(randId)), nil
	}
}

// A filter that inserts 16 bits into the lowest part of the id
/*
			+-----------------------+
			|0|0000.......0000|0...0|
			+-----------------------+
1bit reserved  47bit incr id    16bit random
				1.4 trillion
*/
func With2BytesRandomFilter() Option {
	return func(idgen *IdGenerator) {
		f := AddRandomFilter(16)
		idgen.filters = append(idgen.filters, f)
	}
}
