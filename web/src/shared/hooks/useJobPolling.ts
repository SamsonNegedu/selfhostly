import { useQuery } from '@tanstack/react-query'
import { apiClient } from '../lib/api-client'
import type { Job } from '../types/api'

/**
 * Hook to poll a job's status until completion
 * Automatically stops polling when job reaches completed/failed status
 */
export function useJobPolling(
  jobId: string | null,
  nodeId: string | null,
  enabled: boolean = true
) {
  return useQuery<Job>({
    queryKey: ['job', jobId, nodeId],
    queryFn: () => apiClient.get<Job>(`/api/jobs/${jobId}`, { node_id: nodeId! }),
    enabled: enabled && !!jobId && !!nodeId,
    refetchInterval: (query) => {
      // Stop polling when job is completed or failed
      const data = query.state.data
      if (!data) return 2000
      if (data.status === 'completed' || data.status === 'failed') {
        return false // Stop polling
      }
      return 2000 // Poll every 2 seconds
    },
    retry: 3, // Retry on network errors
    retryDelay: 1000, // Wait 1s between retries
  })
}
