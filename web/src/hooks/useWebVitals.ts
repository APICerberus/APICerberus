import { useEffect, useRef } from 'react'

export interface WebVitalsMetrics {
  lcp: number | null // Largest Contentful Paint (ms)
  fid: number | null // First Input Delay (ms)
  cls: number | null // Cumulative Layout Shift
  ttfb: number | null // Time to First Byte (ms)
  inp: number | null // Interaction to Next Paint (ms)
  fcp: number | null // First Contentful Paint (ms)
}

export interface WebVitalsReport {
  url: string
  timestamp: number
  metrics: WebVitalsMetrics
  rating: 'good' | 'needs-improvement' | 'poor'
}

// Thresholds for Core Web Vitals ratings (based on Google standards)
const THRESHOLDS = {
  lcp: { good: 2500, poor: 4000 },
  fid: { good: 100, poor: 300 },
  cls: { good: 0.1, poor: 0.25 },
  ttfb: { good: 800, poor: 1800 },
  inp: { good: 200, poor: 500 },
  fcp: { good: 1800, poor: 3000 },
}

function getRating(metric: keyof typeof THRESHOLDS, value: number): 'good' | 'needs-improvement' | 'poor' {
  const threshold = THRESHOLDS[metric]
  if (value <= threshold.good) return 'good'
  if (value <= threshold.poor) return 'needs-improvement'
  return 'poor'
}

function getOverallRating(metrics: WebVitalsMetrics): 'good' | 'needs-improvement' | 'poor' {
  const ratings: ('good' | 'needs-improvement' | 'poor')[] = []

  if (metrics.lcp !== null) ratings.push(getRating('lcp', metrics.lcp))
  if (metrics.fid !== null) ratings.push(getRating('fid', metrics.fid))
  if (metrics.cls !== null) ratings.push(getRating('cls', metrics.cls))

  if (ratings.includes('poor')) return 'poor'
  if (ratings.includes('needs-improvement')) return 'needs-improvement'
  return 'good'
}

export function useWebVitals(onReport?: (report: WebVitalsReport) => void) {
  const metricsRef = useRef<WebVitalsMetrics>({
    lcp: null,
    fid: null,
    cls: null,
    ttfb: null,
    inp: null,
    fcp: null,
  })

  useEffect(() => {
    // Check if Performance Observer API is available
    if (typeof window === 'undefined' || !('PerformanceObserver' in window)) {
      return
    }

    const report = () => {
      const reportData: WebVitalsReport = {
        url: window.location.href,
        timestamp: Date.now(),
        metrics: { ...metricsRef.current },
        rating: getOverallRating(metricsRef.current),
      }
      onReport?.(reportData)
    }

    // Observe Largest Contentful Paint (LCP)
    const lcpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries()
      const lastEntry = entries[entries.length - 1] as PerformanceEntry & { startTime: number }
      if (lastEntry) {
        metricsRef.current.lcp = lastEntry.startTime
      }
    })

    // Observe First Input Delay (FID)
    const fidObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries()
      for (const entry of entries) {
        if ('processingStart' in entry) {
          const fidEntry = entry as PerformanceEntry & { processingStart: number; startTime: number }
          metricsRef.current.fid = fidEntry.processingStart - fidEntry.startTime
        }
      }
    })

    // Observe Cumulative Layout Shift (CLS)
  let clsValue = 0
    let clsEntries: PerformanceEntry[] = []

    const clsObserver = new PerformanceObserver((entryList) => {
      for (const entry of entryList.getEntries()) {
        if ('hadRecentInput' in entry && !(entry as { hadRecentInput: boolean }).hadRecentInput) {
          clsEntries.push(entry)
          clsValue += (entry as { value: number; hadRecentInput: boolean }).value
        }
      }
      metricsRef.current.cls = clsValue
    })

    // Observe Time to First Byte (TTFB)
    const ttfbObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries()
      const navigation = entries[0] as PerformanceNavigationTiming
      if (navigation && navigation.responseStart) {
        metricsRef.current.ttfb = navigation.responseStart - navigation.requestStart
      }
    })

    // Observe First Contentful Paint (FCP)
    const fcpObserver = new PerformanceObserver((entryList) => {
      const entries = entryList.getEntries()
      const fcpEntry = entries.find((e) => e.name === 'first-contentful-paint')
      if (fcpEntry) {
        metricsRef.current.fcp = fcpEntry.startTime
      }
    })

    // Start observing
    try {
      lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true })
      fidObserver.observe({ type: 'first-input', buffered: true })
      clsObserver.observe({ type: 'layout-shift', buffered: true })
      ttfbObserver.observe({ type: 'navigation', buffered: true })
      fcpObserver.observe({ type: 'paint', buffered: true })
    } catch {
      // Some metrics may not be supported
    }

    // Report after page is fully loaded (with some buffer for late metrics)
    const reportTimeout = setTimeout(report, 3000)

    // Also report on page unload
    const handleUnload = () => {
      clearTimeout(reportTimeout)
      report()
    }

    window.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        handleUnload()
      }
    })

    return () => {
      clearTimeout(reportTimeout)
      lcpObserver.disconnect()
      fidObserver.disconnect()
      clsObserver.disconnect()
      ttfbObserver.disconnect()
      fcpObserver.disconnect()
      window.removeEventListener('visibilitychange', handleUnload)
    }
  }, [onReport])

  return metricsRef.current
}

// Helper to log vitals to console in development
export function useWebVitalsDev() {
  const { lcp, fid, cls, ttfb, fcp } = useWebVitals((report) => {
    if (import.meta.env.DEV) {
      console.group('[Core Web Vitals]')
      console.log('LCP (Largest Contentful Paint):', report.metrics.lcp?.toFixed(0) + 'ms')
      console.log('FID (First Input Delay):', report.metrics.fid?.toFixed(0) + 'ms')
      console.log('CLS (Cumulative Layout Shift):', report.metrics.cls?.toFixed(3))
      console.log('TTFB (Time to First Byte):', report.metrics.ttfb?.toFixed(0) + 'ms')
      console.log('FCP (First Contentful Paint):', report.metrics.fcp?.toFixed(0) + 'ms')
      console.log('Overall Rating:', report.rating)
      console.groupEnd()
    }
  })
  return { lcp, fid, cls, ttfb, fcp }
}
