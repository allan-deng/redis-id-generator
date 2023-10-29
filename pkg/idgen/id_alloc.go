package idgen

import (
	"fmt"
	"sync"
	"time"
)

type preloadFunc func(bizTag string) (*Seg, error)

type idAllocator struct {
	Key          string       // 'bizTag' is used to distinguish businesses
	Step         int64        // step
	currentPos   int64        // The segment buffer index currently in use; There are two buffer buffers in total, which are recycled
	Buffer       []*idSegment // Two buffers ,One serves as a precache
	UpdateTime   time.Time    // Record the update time to clear the memory when it has not been used for a long time
	mutex        sync.Mutex
	IsPreload    bool // it is being preloaded
	preloadMutex sync.Mutex
	IsInit       bool
	Waiting      []chan byte
}

func NewidAllocator(bizTag string) *idAllocator {
	idAlloc := &idAllocator{
		Key:        bizTag,
		Step:       0,
		currentPos: 0,
		Buffer:     make([]*idSegment, 2),
		UpdateTime: time.Now(),
		IsPreload:  false,
		IsInit:     false,
		Waiting:    make([]chan byte, 0),
	}
	return idAlloc
}

func (this *idAllocator) Init(seg *Seg) {
	this.Lock()
	defer this.Unlock()
	if this.IsInit {
		return
	}

	this.Buffer[0] = newIdSegment(0, 0)
	this.Buffer[1] = newIdSegment(0, 0)

	this.Step = seg.Step
	curSeg := this.getSegment()
	curSeg.init(seg)

	this.IsInit = true
}

func (this *idAllocator) NextId(f preloadFunc) (int64, error) {
	this.Lock()
	defer this.Unlock()

	seg := this.getSegment()
	if !seg.isInit {
		panic(fmt.Sprintf("The [%s] id segment is not initialized", this.Key))
	}

	// get id
	id := seg.getNext()
	this.update()

	// check preload
	if !this.nextSegInited() && seg.needPreLoad() {
		this.preload(f)
	}

	if seg.full() && this.nextSegInited() {
		this.switchSeg()
	}

	if id != 0 {
		return id, nil
	}

	// wait preload..
	// Failure to obtain 'id', usually the current cache 'seg' has been used up,
	// and the next 'seg' has not been initialized.this time should be in preload.
	// Wait for the preload to complete.

	waitChan := make(chan byte, 1)
	this.Waiting = append(this.Waiting, waitChan)
	this.Unlock() // Other requests enter a timed wait
	// Attention：
	// When the step configuration is inappropriate (for example, if the step
	// configuration is too small, the 'seg' is frequently pulled), there may be
	// a large number of requests waiting for the next 'seg'.
	// At this point, if the concurrency is high or the pull seg operation is slow,
	// there may be a large number of requests waiting.
	// Some requests may not get an id because the next seq has not been obtained
	// after waiting for a timeout

	// Wait up to 1000ms
	// The failure is returned promptly and the caller tries again.
	timer := time.NewTimer(2000 * time.Millisecond)
	select {
	case <-waitChan:
	case <-timer.C:
	}

	this.Lock()

	// Check whether the next 'seg' initialization is complete.
	// When finished, need to switch 'seg'.

	if !this.nextSegInited() {
		return 0, fmt.Errorf("[%s]next seg not initialized", this.Key)
	}

	this.switchSeg()
	seg = this.getSegment()

	// FIXME When a request peak is encountered, the number of requests in the waiting state exceeds 'step'.
	// After switching 'seg', there may not be able to assign an 'id' after switching 'seg'.
	// An error is returned and the caller tries again.

	id = seg.getNext()
	this.update()

	if seg.full() {
		return 0, fmt.Errorf("[%s]seg full", this.Key)
	}

	if id == 0 {
		return 0, fmt.Errorf("[%s]allocate id fail", this.Key)
	}

	return id, nil
}

func (this *idAllocator) preload(f preloadFunc) {
	if this.IsPreload {
		return
	}

	go func() {
		// Only one goroutine is preloading at a time
		this.preloadLock()
		defer this.preloadUnLock()

		if this.IsPreload {
			return
		}

		this.IsPreload = true
		nextPos := this.getNextPos()

		segConf, err := f(this.Key)
		if err != nil {
			// TODO need record preload err
			return
		}

		if segConf == nil {
			return
		}

		nowNextPos := this.getNextPos()

		// The next 'seg' can be initialized only if
		// the 'seg switch' does not appear during preload.
		if nextPos == nowNextPos {
			this.getNextSegment().init(segConf)
			this.wakeup()
		}

		this.IsPreload = false
	}()
}

func (this *idAllocator) wakeup() {
	this.Lock()
	defer this.Unlock()
	for _, waitChan := range this.Waiting {
		close(waitChan)
	}
	this.Waiting = this.Waiting[:0]
}

func (this *idAllocator) getNextPos() int64 {
	return (^this.currentPos) & 1
}

func (this *idAllocator) switchSeg() {
	seg := this.getSegment()
	seg.clear()
	this.currentPos = this.getNextPos()
}

func (this *idAllocator) Lock() {
	this.mutex.Lock()
}

func (this *idAllocator) Unlock() {
	this.mutex.Unlock()
}

func (this *idAllocator) preloadLock() {
	this.preloadMutex.Lock()
}

func (this *idAllocator) preloadUnLock() {
	this.preloadMutex.Unlock()
}

func (this *idAllocator) update() {
	this.UpdateTime = time.Now()
}

func (this *idAllocator) getSegment() *idSegment {
	return this.Buffer[this.currentPos]
}

func (this *idAllocator) getNextSegment() *idSegment {
	return this.Buffer[this.getNextPos()]
}

func (this *idAllocator) nextSegInited() bool {
	return this.Buffer[this.getNextPos()].isInit
}
