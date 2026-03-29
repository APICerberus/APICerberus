package ratelimit

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type fixedWindowState struct {
	mu       sync.Mutex
	windowID atomic.Int64
	count    atomic.Int64
}

// FixedWindow is an in-memory fixed-window limiter keyed by scope key.
type FixedWindow struct {
	limit         int64
	windowSeconds int64
	windows       sync.Map // map[string]*fixedWindowState
	now           func() time.Time
}

// NewFixedWindow creates fixed window limiter.
func NewFixedWindow(limit int, window time.Duration) *FixedWindow {
	if limit <= 0 {
		limit = 1
	}
	windowSeconds := int64(window / time.Second)
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	return &FixedWindow{
		limit:         int64(limit),
		windowSeconds: windowSeconds,
		now:           time.Now,
	}
}

// Allow increments counter for key in current window and returns decision/remaining/resetAt.
func (fw *FixedWindow) Allow(key string) (allowed bool, remaining int, resetAt time.Time) {
	if fw == nil {
		return false, 0, time.Time{}
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "_global"
	}

	now := fw.now()
	windowID := fw.windowID(now)
	resetAt = time.Unix((windowID+1)*fw.windowSeconds, 0)

	raw, _ := fw.windows.LoadOrStore(key, &fixedWindowState{})
	state := raw.(*fixedWindowState)
	fw.ensureWindow(state, windowID)

	count := state.count.Add(1)
	if count <= fw.limit {
		allowed = true
		remaining = int(fw.limit - count)
		if remaining < 0 {
			remaining = 0
		}
		return allowed, remaining, resetAt
	}
	return false, 0, resetAt
}

func (fw *FixedWindow) windowID(ts time.Time) int64 {
	return ts.Unix() / fw.windowSeconds
}

func (fw *FixedWindow) ensureWindow(state *fixedWindowState, currentWindowID int64) {
	if state.windowID.Load() == currentWindowID {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.windowID.Load() == currentWindowID {
		return
	}
	state.windowID.Store(currentWindowID)
	state.count.Store(0)
}
