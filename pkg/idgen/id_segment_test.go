package idgen

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_idSegment_getNext(t *testing.T) {
	assert := assert.New(t)

	idSeg := &idSegment{
		Max:    math.MaxInt64,
		Min:    0,
		Cur:    0,
		isInit: true,
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10000; j++ {
				idSeg.getNext()
			}
			wg.Done()
		}()
	}
	wg.Wait()

	assert.Equal(idSeg.Cur, int64(10000*10000), "Concurrent get.")

	idSeg.Max = 0
	want := idSeg.getNext()

	assert.Equal(want, int64(0), "Out of seg boundary.")
}
