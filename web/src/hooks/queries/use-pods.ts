import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Pod } from '@/api/types'

export function usePods(refetchInterval = 5000) {
  return useQuery({
    queryKey: ['pods'],
    queryFn: () => apiClient.get<Pod[]>('/pods').then((r) => r.data),
    refetchInterval,
  })
}

export function usePod(podId: string) {
  return useQuery({
    queryKey: ['pods', podId],
    queryFn: () => apiClient.get<Pod>(`/pods/${podId}`).then((r) => r.data),
    enabled: !!podId,
  })
}
