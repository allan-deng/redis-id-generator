package idgen

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"
)

const (
	MaxStep               = 10e6
	DefaultStep           = 2000            // default step
	DefaultRetry          = 3               // default retry times when id allocator preload next segment
	DefaultPreloadTimeout = 3 * time.Second // default timeout  when id allocator preload next segment
)

type IdGenerator struct {
	cache      *bizCache     // Cache that each biztag has obtained segments on the local machine that can be used for allocation
	expireTime time.Duration // Cache expire time. When it is 0, it will never expire
	step       int64         // When no biztag info is stored, the initialization is done using step

	store IdStore

	preloadRetryTimes int
	preloadTimeout    time.Duration

	// Filter for Post-processing of 'ID'.
	// Due to the use of 'ID segmentation', the obtained raw IDs are sequentially incremented,
	// which may be maliciously exploited for traversal. If there is a need to obfuscate the
	// IDs, filters can be used.
	filters []IdFilter
}

type Option func(*IdGenerator)

func WithExpireTime(expireTime time.Duration) Option {
	return func(idgen *IdGenerator) {
		idgen.expireTime = expireTime
	}
}

func WithPreloadTimeout(timeout time.Duration) Option {
	return func(idgen *IdGenerator) {
		idgen.preloadTimeout = timeout
	}
}

func WithPreloadRetryTimes(times int) Option {
	return func(idgen *IdGenerator) {
		idgen.preloadRetryTimes = times
	}
}

func WithStep(step int64) Option {
	if step > MaxStep {
		step = MaxStep
	}
	return func(idgen *IdGenerator) {
		idgen.step = step
	}
}

func WithIdFilter(filters []IdFilter) Option {
	return func(idgen *IdGenerator) {
		idgen.filters = append(idgen.filters, filters...)
	}
}

func NewIdGenrator(store IdStore, opts ...Option) *IdGenerator {
	idGen := &IdGenerator{
		store:             store,
		preloadRetryTimes: DefaultRetry,
		expireTime:        0,
		step:              DefaultStep,
		preloadTimeout:    DefaultPreloadTimeout,
	}

	for _, o := range opts {
		o(idGen)
	}

	idGen.cache = newBizCache(idGen.expireTime)
	return idGen
}

func (this *IdGenerator) GetId(ctx context.Context, bizTag string) (int64, error) {
	// Find id allocator through bizTag
	idAlloc := this.cache.get(bizTag)
	if idAlloc == nil {
		// Create and initialize an id allocator and add it to the cache
		var err error
		idAlloc, err = this.AddBizTag(ctx, bizTag)
		if err != nil {
			return 0, err
		}
	}

	ctxPreload, _ := context.WithTimeout(context.Background(), this.preloadTimeout)

	// Specifies the function used for preload in allocator
	preload := func(bizTag string) (*Seg, error) {
		var seg *Seg
		var err error
		for i := 0; i <= this.preloadRetryTimes; i++ {
			seg, err = this.store.GetNextSegment(ctxPreload, bizTag, this.step)
			if err == nil {
				return seg, nil
			}
		}
		return seg, err
	}

	id, err := idAlloc.NextId(preload)
	if err != nil {
		return 0, err
	}

	for _, f := range this.filters {
		id, err = f(id)
		if err != nil {
			filterName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
			return 0, fmt.Errorf("id filter failed. func[%s], id[%d]", filterName, id)
		}
	}
	return id, err

}

func (this *IdGenerator) AddBizTag(ctx context.Context, bizTag string) (*idAllocator, error) {
	// FIXME When concurrent add occurs, the creation and initialization will be repeated.

	seg, err := this.store.GetNextSegment(ctx, bizTag, this.step)
	if err != nil {
		return nil, err
	}

	// Create and initialize
	idAlloc := NewidAllocator(bizTag)
	idAlloc.Init(seg)
	this.cache.add(idAlloc)
	idAlloc.update()

	return idAlloc, nil
}
