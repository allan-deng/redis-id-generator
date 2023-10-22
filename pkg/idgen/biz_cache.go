package idgen

import (
	"sync"
	"time"
)

type bizCache struct {
	cache      sync.Map
	expireTime time.Duration // If the expireTime is not 0, the expired cache is periodically cleared
}

func newBizCache(expireTime time.Duration) *bizCache {
	cache := &bizCache{
		expireTime: expireTime,
	}

	if cache.expireTime != 0 {
		go cache.clear()
	}

	return cache
}

func (this *bizCache) get(bizTag string) *idAllocator {
	if alloc, ok := this.cache.Load(bizTag); ok {
		return alloc.(*idAllocator)
	}
	return nil
}

func (this *bizCache) add(idAlloc *idAllocator) {
	this.cache.Store(idAlloc.Key, idAlloc)
}

func (this *bizCache) clear() {
	for {
		now := time.Now()
		next := now.Add(this.expireTime)
		t := time.NewTimer(next.Sub(now))
		<-t.C

		this.cache.Range(func(key, value interface{}) bool {
			alloc := value.(*idAllocator)
			if next.Sub(alloc.UpdateTime) > this.expireTime {
				// TODO need record clear operation
				this.cache.Delete(key)
			}
			return true
		})
	}
}
