package idgen

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type storeDemo struct {
	lock sync.Mutex
	maxs map[string]int64
	once sync.Once
}

func (s *storeDemo) GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error) {
	s.once.Do(func() {
		s.maxs = make(map[string]int64)
	})

	s.lock.Lock()
	defer s.lock.Unlock()

	max, ok := s.maxs[bizTag]
	if !ok {
		max = 0
	}
	max += step
	s.maxs[bizTag] = max
	return &Seg{
		BizTag: bizTag,
		MaxId:  max,
		Step:   step,
	}, nil
}

type storeFastDemo struct {
	max int64
}

func (s *storeFastDemo) GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error) {

	newMax := atomic.AddInt64(&s.max, step)
	return &Seg{
		BizTag: bizTag,
		MaxId:  newMax,
		Step:   step,
	}, nil
}

func TestWithExpireTime(t *testing.T) {
	assert := assert.New(t)

	store := storeDemo{}

	tests := []struct {
		name       string
		expireTime time.Duration
	}{
		{
			name:       "5ms",
			expireTime: 5 * time.Millisecond,
		},
		{
			name:       "50ms",
			expireTime: 50 * time.Millisecond,
		},
		{
			name:       "500ms",
			expireTime: 500 * time.Millisecond,
		},
		{
			name:       "5000ms",
			expireTime: 5000 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idGen := NewIdGenrator(&store, WithExpireTime(tt.expireTime))
			id, err := idGen.GetId(context.Background(), "test")

			assert.Nil(err, "get id err")
			assert.NotEqual(int64(0), id, "id is zero")
			assert.NotNil(idGen.cache.get("test"), "biztag not in cache")

			time.Sleep(tt.expireTime + 1000*time.Millisecond)
			assert.Nil(idGen.cache.get("test"), "biztag not expired: %d ms", tt.expireTime.Milliseconds())
		})
	}

}

func TestWithStep(t *testing.T) {
	assert := assert.New(t)

	store := storeDemo{}

	tests := []struct {
		name string
		step int64
		want int64
	}{
		{
			name: "normal",
			step: 1000,
			want: 1000,
		},
		{
			name: "zero",
			step: 0,
			want: DefaultStep,
		},
		{
			name: "negative",
			step: -1,
			want: DefaultStep,
		},
		{
			name: "min int 64",
			step: math.MinInt64,
			want: DefaultStep,
		},
		{
			name: "max int 64",
			step: math.MaxInt64,
			want: MaxStep,
		},
		{
			name: "beyond max step",
			step: MaxStep + 1,
			want: MaxStep,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idGen := NewIdGenrator(&store, WithStep(tt.step))
			assert.Equal(tt.want, idGen.step, "step err, set step: %d", tt.step)
		})
	}
}

func TestIdGenerator_GetId(t *testing.T) {
	assert := assert.New(t)

	var bizTagMap sync.Map

	var step int64 = 200000
	var tags int = 50
	var goroutindes int = 50
	var timesPerGoroutine int = 1000

	store := storeDemo{}
	idGen := NewIdGenrator(&store, WithStep(step))

	wg := sync.WaitGroup{}
	for i := 0; i < tags; i++ {
		bizTag := fmt.Sprintf("test-%d", i)
		var res sync.Map
		bizTagMap.Store(bizTag, &res)

		for j := 0; j < goroutindes; j++ {
			wg.Add(1)
			go func() {
				for j := 0; j < timesPerGoroutine; j++ {
					id, err := idGen.GetId(context.Background(), bizTag)

					// test: Concurrent id generation without error
					assert.Nil(err, "Concurrent get next id failed. err: %s", func() string {
						if err != nil {
							return err.Error()
						}
						return ""
					}())

					// test: No duplicate id is generated concurrently
					resMap, ok := bizTagMap.Load(bizTag)
					assert.Equal(true, ok, "not found res store map. bizTag: %s", bizTag)
					_, loaded := resMap.(*sync.Map).LoadOrStore(id, 1)
					assert.Equal(false, loaded, "id duplication: %d, bizTag: %s", id, bizTag)
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()
}

func BenchmarkGetId(b *testing.B) {
	store := storeFastDemo{}
	var step int64 = MaxStep
	idGen := NewIdGenrator(&store, WithStep(step))
	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		_, err := idGen.GetId(ctx, "test")
		if err != nil {
			b.Errorf("get id err: %s", err)
		}
	}
}
