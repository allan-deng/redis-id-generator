package idgen

import (
	"context"
)

type Seg struct {
	BizTag string
	MaxId  int64
	Step   int64
}

type IdStore interface {
	GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error)
}
