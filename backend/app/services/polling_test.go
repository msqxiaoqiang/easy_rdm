package services

import (
	"sync"
	"testing"
)

func TestEventBuffer_PushAndSince(t *testing.T) {
	buf := NewEventBuffer(100)

	buf.Push("test", map[string]string{"msg": "hello"})
	buf.Push("test", map[string]string{"msg": "world"})

	events := buf.Since(0)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("unexpected seq: %d, %d", events[0].Seq, events[1].Seq)
	}
	if events[0].Type != "test" {
		t.Fatalf("expected type 'test', got %s", events[0].Type)
	}
}

func TestEventBuffer_SinceIncremental(t *testing.T) {
	buf := NewEventBuffer(100)

	buf.Push("a", "data1")
	buf.Push("b", "data2")
	buf.Push("c", "data3")

	// 获取 seq=1 之后的事件
	events := buf.Since(1)
	if len(events) != 2 {
		t.Fatalf("expected 2 events after seq 1, got %d", len(events))
	}
	if events[0].Seq != 2 {
		t.Fatalf("expected first event seq=2, got %d", events[0].Seq)
	}
}

func TestEventBuffer_SinceLatest(t *testing.T) {
	buf := NewEventBuffer(100)
	buf.Push("a", "data1")
	buf.Push("b", "data2")

	events := buf.Since(2)
	if events != nil {
		t.Fatalf("expected nil for latest seq, got %d events", len(events))
	}
}

func TestEventBuffer_MaxLen(t *testing.T) {
	buf := NewEventBuffer(3)

	for i := 0; i < 5; i++ {
		buf.Push("test", i)
	}

	events := buf.Since(0)
	if len(events) != 3 {
		t.Fatalf("expected 3 events (maxLen), got %d", len(events))
	}
	// 最早的应该是 seq=3（前两个被丢弃）
	if events[0].Seq != 3 {
		t.Fatalf("expected first event seq=3, got %d", events[0].Seq)
	}
}

func TestEventBuffer_Clear(t *testing.T) {
	buf := NewEventBuffer(100)
	buf.Push("test", "data")
	buf.Clear()

	events := buf.Since(0)
	if events != nil {
		t.Fatalf("expected nil after clear, got %d events", len(events))
	}
}

func TestEventBuffer_ConcurrentSafety(t *testing.T) {
	buf := NewEventBuffer(1000)
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Push("test", n*100+j)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Since(0)
			}
		}()
	}

	wg.Wait()

	events := buf.Since(0)
	if len(events) != 1000 {
		t.Fatalf("expected 1000 events, got %d", len(events))
	}
}

func TestGetBuffer_CreateAndReuse(t *testing.T) {
	// 清理全局状态
	buffersMu.Lock()
	buffers = make(map[string]*EventBuffer)
	buffersMu.Unlock()

	buf1 := GetBuffer("conn1", "pubsub", 100)
	buf2 := GetBuffer("conn1", "pubsub", 100)

	if buf1 != buf2 {
		t.Fatal("GetBuffer should return same instance for same key")
	}

	buf3 := GetBuffer("conn1", "monitor", 100)
	if buf1 == buf3 {
		t.Fatal("different scenes should have different buffers")
	}
}

func TestRemoveBuffers(t *testing.T) {
	buffersMu.Lock()
	buffers = make(map[string]*EventBuffer)
	buffersMu.Unlock()

	GetBuffer("conn1", "pubsub", 100)
	GetBuffer("conn1", "monitor", 100)
	GetBuffer("conn2", "pubsub", 100)

	RemoveBuffers("conn1")

	buffersMu.RLock()
	defer buffersMu.RUnlock()

	if _, ok := buffers["conn1:pubsub"]; ok {
		t.Fatal("conn1:pubsub should be removed")
	}
	if _, ok := buffers["conn1:monitor"]; ok {
		t.Fatal("conn1:monitor should be removed")
	}
	if _, ok := buffers["conn2:pubsub"]; !ok {
		t.Fatal("conn2:pubsub should still exist")
	}
}
