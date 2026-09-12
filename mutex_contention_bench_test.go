package bench

import (
	"sync"
	"sync/atomic"
	"testing"
)

const numShards = 16

// Approach 1: one mutex, one counter. Every writer serializes here. 

type SingleMutex struct {
	mu    sync.Mutex
	count int64
}

func (c *SingleMutex) Inc() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

//  Approach 2: N independently-locked shards. A goroutine only
// contends with whoever else lands on its shard. 

type Sharded struct {
	shards [numShards]struct {
		mu    sync.Mutex
		count int64
		_     [40]byte // pad to 64B so shards don't share a cache line
	}
}

func (c *Sharded) Inc(shard int) {
	s := &c.shards[shard%numShards]
	s.mu.Lock()
	s.count++
	s.mu.Unlock()
}

//  Approach 3: no lock at all. 

type Atomic struct {
	count int64
}

func (c *Atomic) Inc() {
	atomic.AddInt64(&c.count, 1)
}

func BenchmarkSingleMutex(b *testing.B) {
	c := &SingleMutex{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkSharded(b *testing.B) {
	c := &Sharded{}
	var next int64
	b.RunParallel(func(pb *testing.PB) {
		shard := int(atomic.AddInt64(&next, 1))
		for pb.Next() {
			c.Inc(shard)
		}
	})
}

func BenchmarkAtomic(b *testing.B) {
	c := &Atomic{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// Bonus: RWMutex only pays off once reads actually dominate.

func BenchmarkMutexRead(b *testing.B) {
	var mu sync.Mutex
	value := 42
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			_ = value
			mu.Unlock()
		}
	})
}

func BenchmarkRWMutexRead(b *testing.B) {
	var mu sync.RWMutex
	value := 42
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			_ = value
			mu.RUnlock()
		}
	})
}