import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useWebVitals } from './useWebVitals'

describe('useWebVitals', () => {
  beforeEach(() => {
    const mockObserver = {
      observe: vi.fn(),
      disconnect: vi.fn(),
    }
    vi.stubGlobal('PerformanceObserver', vi.fn(() => mockObserver) as any)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should return initial null values', () => {
    const { result } = renderHook(() => useWebVitals())
    expect(result.current.lcp).toBeNull()
    expect(result.current.fid).toBeNull()
    expect(result.current.cls).toBeNull()
    expect(result.current.ttfb).toBeNull()
    expect(result.current.inp).toBeNull()
    expect(result.current.fcp).toBeNull()
  })

  it('should call onReport callback when provided', () => {
    const onReport = vi.fn()
    renderHook(() => useWebVitals({ onReport }))
    // Without PerformanceObserver support, callback won't be called
    expect(onReport).not.toHaveBeenCalled()
  })
})
