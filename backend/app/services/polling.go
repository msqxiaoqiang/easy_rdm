package services

import (
	"encoding/json"
	"sync"
	"time"
)

// PollEvent 长轮询事件
type PollEvent struct {
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	Timestamp int64           `json:"ts"`
	Data      json.RawMessage `json:"data"`
}

// EventBuffer 事件缓冲区（按连接+场景隔离）
type EventBuffer struct {
	mu       sync.RWMutex
	events   []PollEvent
	seq      int64
	maxLen   int
	notifyCh chan struct{} // 有新事件时通知等待者
}

// NewEventBuffer 创建事件缓冲区
func NewEventBuffer(maxLen int) *EventBuffer {
	return &EventBuffer{
		events:   make([]PollEvent, 0, maxLen),
		maxLen:   maxLen,
		notifyCh: make(chan struct{}, 1),
	}
}

// Push 追加事件
func (b *EventBuffer) Push(eventType string, data interface{}) {
	raw, _ := json.Marshal(data)
	b.mu.Lock()
	b.seq++
	evt := PollEvent{
		Seq:       b.seq,
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		Data:      raw,
	}
	b.events = append(b.events, evt)
	if len(b.events) > b.maxLen {
		b.events = b.events[len(b.events)-b.maxLen:]
	}
	b.mu.Unlock()
	// 非阻塞通知等待者
	select {
	case b.notifyCh <- struct{}{}:
	default:
	}
}

// Since 返回 seq 之后的所有事件
func (b *EventBuffer) Since(afterSeq int64) []PollEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for i, evt := range b.events {
		if evt.Seq > afterSeq {
			result := make([]PollEvent, len(b.events)-i)
			copy(result, b.events[i:])
			return result
		}
	}
	return nil
}

// WaitSince 阻塞等待 afterSeq 之后的事件，超时返回空
func (b *EventBuffer) WaitSince(afterSeq int64, timeout time.Duration) []PollEvent {
	// 先检查是否已有数据
	if events := b.Since(afterSeq); len(events) > 0 {
		return events
	}
	// 无数据则等待通知或超时
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-b.notifyCh:
		return b.Since(afterSeq)
	case <-timer.C:
		return nil
	}
}

// Clear 清空缓冲区
func (b *EventBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = b.events[:0]
}

// ========== 全局事件管理 ==========

var (
	buffersMu sync.RWMutex
	// key: "{connID}:{scene}" 如 "abc123:pubsub", "abc123:monitor"
	buffers = make(map[string]*EventBuffer)
)

// GetBuffer 获取或创建指定场景的事件缓冲区
func GetBuffer(connID, scene string, maxLen int) *EventBuffer {
	key := connID + ":" + scene
	buffersMu.RLock()
	buf, ok := buffers[key]
	buffersMu.RUnlock()
	if ok {
		return buf
	}

	buffersMu.Lock()
	defer buffersMu.Unlock()
	// 双重检查
	if buf, ok = buffers[key]; ok {
		return buf
	}
	buf = NewEventBuffer(maxLen)
	buffers[key] = buf
	return buf
}

// RemoveBuffers 移除指定连接的所有缓冲区
func RemoveBuffers(connID string) {
	buffersMu.Lock()
	defer buffersMu.Unlock()
	prefix := connID + ":"
	for key := range buffers {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(buffers, key)
		}
	}
}
