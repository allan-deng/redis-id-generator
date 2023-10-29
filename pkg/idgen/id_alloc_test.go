package idgen

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_idAllocator_getNextPos(t *testing.T) {

	f := func(pos int64) idAllocator {
		idAlloc := NewidAllocator("test")
		idAlloc.currentPos = pos
		return *idAlloc
	}
	tests := []struct {
		name    string
		idAlloc idAllocator
		want    int64
	}{
		{
			name:    "test pos 0",
			idAlloc: f(0),
			want:    1,
		},
		{
			name:    "test pos 1",
			idAlloc: f(1),
			want:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.idAlloc.getNextPos(); got != tt.want {
				t.Errorf("idAllocator.getNextPos() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_idAllocator_preload(t *testing.T) {

	assert := assert.New(t)

	type args struct {
		f preloadFunc
	}
	getIdAlloc := func(isPreload bool) idAllocator {
		idAlloc := NewidAllocator("test")
		idAlloc.Init(&Seg{
			BizTag: "test",
			MaxId:  2000,
			Step:   2000,
		})
		idAlloc.IsPreload = isPreload
		return *idAlloc
	}

	succPreloadFunc := func(bizTag string) (*Seg, error) {
		return &Seg{
			BizTag: "test",
			MaxId:  2000,
			Step:   2000,
		}, nil
	}
	failPreloadFunc := func(bizTag string) (*Seg, error) {
		return nil, errors.New("preload err")
	}

	tests := []struct {
		name       string
		idAlloc    idAllocator
		args       preloadFunc
		assertFunc func(idAlloc idAllocator)
	}{
		{
			name:    "test. in another preload process",
			idAlloc: getIdAlloc(true),
			args:    succPreloadFunc,
			assertFunc: func(idAlloc idAllocator) {
				assert.Equal(idAlloc.getNextSegment().isInit, false, "test. preload func return err")
			},
		},
		{
			name:    "test. preload func return err",
			idAlloc: getIdAlloc(false),
			args:    failPreloadFunc,
			assertFunc: func(idAlloc idAllocator) {
				assert.Equal(idAlloc.getNextSegment().isInit, false, "test. preload func return err")

			},
		},
		{
			name:    "test. succ",
			idAlloc: getIdAlloc(false),
			args:    succPreloadFunc,
			assertFunc: func(idAlloc idAllocator) {
				assert.Equal(idAlloc.getNextSegment().isInit, true, "test. succ")

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.idAlloc.preload(tt.args)
			time.Sleep(10 * time.Millisecond)
			tt.assertFunc(tt.idAlloc)
		})
	}
}

func Test_idAllocator_NextId(t *testing.T) {
	assert := assert.New(t)
	var res sync.Map

	var step int64 = 200000
	var maxid int64 = 0
	var goroutines int = 100
	var timesPerGoroutine int = 100000

	getNextSeg := func(bizTag string) (*Seg, error) {
		newMaxId := atomic.AddInt64(&maxid, step)
		return &Seg{
			BizTag: "test",
			MaxId:  newMaxId,
			Step:   step,
		}, nil
	}

	getIdAlloc := func() idAllocator {
		idAlloc := NewidAllocator("test")
		seg, _ := getNextSeg("test")
		idAlloc.Init(seg)
		return *idAlloc
	}

	idAlloc := getIdAlloc()

	wg := sync.WaitGroup{}
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < timesPerGoroutine; j++ {
				id, err := idAlloc.NextId(getNextSeg)

				// test: Concurrent id generation without error
				assert.Nil(err, "Concurrent get next id failed. err: %s", func() string {
					if err != nil {
						return err.Error()
					}
					return ""
				}())

				// test: No duplicate id is generated concurrently
				_, loaded := res.LoadOrStore(id, 1)
				assert.Equal(false, loaded, "id duplication: %d", id)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// test: The number of ids generated is correct
	count := 0
	res.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(goroutines*timesPerGoroutine, count, "id num err: %d", count)
}
