package idgen

import "sync/atomic"

type idSegment struct {
	Max    int64
	Min    int64
	Cur    int64
	isInit bool
}

func newIdSegment(max int64, step int64) *idSegment {
	seg := &idSegment{}
	seg.clear()
	return seg
}

func (this *idSegment) init(conf *Seg) {
	this.Max = conf.MaxId - 1
	this.Min = conf.MaxId - conf.Step
	this.Cur = this.Min
	this.isInit = true
}

func (this *idSegment) GetNext() int64 {
	id := atomic.AddInt64(&this.Cur, 1)
	if id >= this.Max {
		return 0
	}
	return id
}

func (this *idSegment) needPreLoad() bool {
	return this.Max-this.Cur < int64(0.9*float32(this.Max-this.Min))
}

func (this *idSegment) full() bool {
	return this.Cur >= this.Max
}

func (this *idSegment) clear() {
	this.Max = 0
	this.Min = 0
	this.Cur = 0
	this.isInit = false
}
