package idgen

import (
	"fmt"
	"sync"
	"time"
)

type preloadFunc func(bizTag string) (*Seg, error)

type idAllocator struct {
	Key          string       // 也就是`biz_tag`用来区分业务
	Step         int64        // 记录步长
	currentPos   int64        // 当前使用的 segment buffer光标; 总共两个buffer缓存区，循环使用
	Buffer       []*idSegment // 双buffer 一个作为预缓存作用
	UpdateTime   time.Time    // 记录更新时间 方便长时间不用进行清理，防止占用内存
	mutex        sync.Mutex   // 互斥锁
	IsPreload    bool         // 是否正在预加载
	preloadMutex sync.Mutex
	IsInit       bool
	Waiting      []chan byte // 挂起等待
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

	// 判断当前的buf 是否可用
	seg := this.getSegment()
	if !seg.isInit {
		panic(fmt.Sprintf("The [%s] id segment is not initialized", this.Key))
	}

	// 获取 id
	id := seg.GetNext()
	this.update()

	// 判断预加载
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

	// Wait up to 500ms
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

	id = seg.GetNext()
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
	if this.currentPos == 0 {
		return 1
	} else {
		return 0
	}
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
