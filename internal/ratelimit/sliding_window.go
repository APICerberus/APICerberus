package ratelimit

import (
	"math"
	"strings"
	"sync"
	"time"
)

type slidingWindowState struct {
	mu     sync.Mutex
	counts map[int64]int64 // slotID -> count
}

// SlidingWindow enforces rate limits over rolling time windows via sub-window buckets.
type SlidingWindow struct {
	limit      int64
	window     time.Duration
	subWindow  time.Duration
	windowSpan int64
	buckets    sync.Map // map[string]*slidingWindowState
	now        func() time.Time
}

func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Second
	}
	sub := window / 10
	if sub <= 0 {
		sub = 100 * time.Millisecond
	}
	return &SlidingWindow{
		limit:      int64(limit),
		window:     window,
		subWindow:  sub,
		windowSpan: int64(math.Ceil(float64(window) / float64(sub))),
		now:        time.Now,
	}
}

// Allow adds one event for key and returns decision/remaining/resetAt.
func (sw *SlidingWindow) Allow(key string) (allowed bool, remaining int, resetAt time.Time) {
	if sw == nil {
		return false, 0, time.Time{}
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "_global"
	}
	now := sw.now()
	slot := sw.slotID(now)

	raw, _ := sw.buckets.LoadOrStore(key, &slidingWindowState{counts: make(map[int64]int64)})
	state := raw.(*slidingWindowState)

	state.mu.Lock()
	defer state.mu.Unlock()

	sw.pruneLocked(state, slot)

	total := int64(0)
	for _, c := range state.counts {
		total += c
	}

	if total >= sw.limit {
		remaining = 0
		resetAt = sw.nextResetAtLocked(state, now)
		return false, remaining, resetAt
	}

	state.counts[slot]++
	total++
	remaining = int(sw.limit - total)
	if remaining < 0 {
		remaining = 0
	}
	resetAt = sw.nextResetAtLocked(state, now)
	return true, remaining, resetAt
}

func (sw *SlidingWindow) slotID(ts time.Time) int64 {
	return ts.UnixNano() / sw.subWindow.Nanoseconds()
}

func (sw *SlidingWindow) pruneLocked(state *slidingWindowState, currentSlot int64) {
	minSlot := currentSlot - sw.windowSpan + 1
	for slot := range state.counts {
		if slot < minSlot {
			delete(state.counts, slot)
		}
	}
}

func (sw *SlidingWindow) nextResetAtLocked(state *slidingWindowState, now time.Time) time.Time {
	if len(state.counts) == 0 {
		return now
	}
	oldest := int64(^uint64(0) >> 1)
	for slot, count := range state.counts {
		if count <= 0 {
			continue
		}
		if slot < oldest {
			oldest = slot
		}
	}
	if oldest == int64(^uint64(0)>>1) {
		return now
	}
	slotStart := time.Unix(0, oldest*sw.subWindow.Nanoseconds())
	return slotStart.Add(sw.window).Add(sw.subWindow)
}
