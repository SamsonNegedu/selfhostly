import { useEffect, useRef, useState } from 'react'
import type { JobLogLine } from '../types/api'

const DEPLOY_JOB_TYPES = ['app_create', 'app_update', 'app_start'] as const

export function isDeployJobType(type: string): boolean {
  return (DEPLOY_JOB_TYPES as readonly string[]).includes(type)
}

/**
 * Streams deployment log lines via EventSource (SSE), falling back to REST polling on error.
 */
export function useDeploymentLogStream(
  jobId: string | null | undefined,
  nodeId: string | null | undefined,
  enabled: boolean
) {
  const [lines, setLines] = useState<JobLogLine[]>([])
  const afterRef = useRef(0)
  const linesBySeq = useRef<Map<number, string>>(new Map())

  useEffect(() => {
    if (!enabled || !jobId || !nodeId) {
      setLines([])
      linesBySeq.current.clear()
      afterRef.current = 0
      return
    }

    setLines([])
    linesBySeq.current.clear()
    afterRef.current = 0

    let cancelled = false
    const base = `/api/jobs/${encodeURIComponent(jobId)}/logs`
    const qs = `node_id=${encodeURIComponent(nodeId)}`

    const mergeLine = (seq: number, text: string) => {
      if (linesBySeq.current.has(seq)) return
      linesBySeq.current.set(seq, text)
      setLines(
        Array.from(linesBySeq.current.entries())
          .sort((a, b) => a[0] - b[0])
          .map(([seq, t]) => ({ seq, text: t }))
      )
      if (seq > afterRef.current) {
        afterRef.current = seq
      }
    }

    let pollTimer: ReturnType<typeof setInterval> | null = null

    const poll = async () => {
      if (cancelled) return
      try {
        const res = await fetch(`${base}?${qs}&after=${afterRef.current}`, {
          credentials: 'include',
        })
        if (!res.ok) return
        const data = await res.json()
        const payload = data as { lines: JobLogLine[]; next_after: number; job_status: string }
        for (const row of payload.lines ?? []) {
          mergeLine(row.seq, row.text)
        }
        if (typeof payload.next_after === 'number' && payload.next_after > afterRef.current) {
          afterRef.current = payload.next_after
        }
        if (payload.job_status === 'completed' || payload.job_status === 'failed') {
          if (pollTimer) {
            clearInterval(pollTimer)
            pollTimer = null
          }
        }
      } catch {
        /* ignore */
      }
    }

    const startPolling = () => {
      if (pollTimer) return
      void poll()
      pollTimer = setInterval(poll, 800)
    }

    if (typeof EventSource === 'undefined') {
      startPolling()
      return () => {
        cancelled = true
        if (pollTimer) clearInterval(pollTimer)
      }
    }

    const streamUrl = `${base}/stream?${qs}&after=0`
    const es = new EventSource(streamUrl, { withCredentials: true })

    const onLine = (e: MessageEvent) => {
      try {
        const row = JSON.parse(e.data as string) as JobLogLine
        if (typeof row.seq === 'number' && typeof row.text === 'string') {
          mergeLine(row.seq, row.text)
        }
      } catch {
        /* ignore */
      }
    }

    const onDone = () => {
      es.close()
    }

    es.addEventListener('line', onLine)
    es.addEventListener('done', onDone)

    es.onerror = () => {
      es.close()
      es.removeEventListener('line', onLine)
      es.removeEventListener('done', onDone)
      if (!cancelled) {
        startPolling()
      }
    }

    return () => {
      cancelled = true
      es.close()
      es.removeEventListener('line', onLine)
      es.removeEventListener('done', onDone)
      if (pollTimer) clearInterval(pollTimer)
    }
  }, [jobId, nodeId, enabled])

  return { lines }
}
