package tailer

import (
	"sync"
	"time"
)

type Batcher struct {
	mu          sync.Mutex
	items       []LogLine
	maxBatch    int
	interval    time.Duration
	flushFn     func([]LogLine)
	ticker      *time.Ticker
	stopChan    chan struct{}
	stoppedOnce sync.Once
}

func NewBatcher(maxBatch int, interval time.Duration, flushFn func([]LogLine)) *Batcher {
	b := &Batcher{
		items:    make([]LogLine, 0, maxBatch),
		maxBatch: maxBatch,
		interval: interval,
		flushFn:  flushFn,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
	}

	go b.loop()
	return b
}

func (b *Batcher) loop() {
	for {
		select {
		case <-b.ticker.C:
			b.Flush()
		case <-b.stopChan:
			b.ticker.Stop()
			b.Flush()
			return
		}
	}
}

func (b *Batcher) Add(line LogLine) {
	b.mu.Lock()
	b.items = append(b.items, line)
	readyToFlush := len(b.items) >= b.maxBatch
	var toFlush []LogLine
	if readyToFlush {
		toFlush = b.items
		b.items = make([]LogLine, 0, b.maxBatch)
	}
	b.mu.Unlock()

	if readyToFlush && len(toFlush) > 0 {
		b.flushFn(toFlush)
	}
}

func (b *Batcher) Flush() {
	b.mu.Lock()
	if len(b.items) == 0 {
		b.mu.Unlock()
		return
	}
	toFlush := b.items
	b.items = make([]LogLine, 0, b.maxBatch)
	b.mu.Unlock()

	b.flushFn(toFlush)
}

func (b *Batcher) Stop() {
	b.stoppedOnce.Do(func() {
		close(b.stopChan)
	})
}
