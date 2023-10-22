package idgen

import (
	"context"
	"sync/atomic"
)

type Seg struct {
	BizTag string
	MaxId  int64
	Step   int64
}

type IdStore interface {
	GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error)
}

type RedisIdStore struct {
	max int64
}

func NewRedisIdStore() *RedisIdStore {
	return &RedisIdStore{
		max: 0,
	}
}

func (this *RedisIdStore) GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error) {
	// If there is no key in redis, it is created; otherwise, it is updated
	return &Seg{
		BizTag: bizTag,
		MaxId:  atomic.AddInt64(&this.max, step),
		Step:   step,
	}, nil
}
