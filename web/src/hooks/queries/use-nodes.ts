import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Node } from '@/api/types'

export function useNodes(refetchInterval = 5000) {
  return useQuery({
    queryKey: ['nodes'],
    queryFn: () => apiClient.get<Node[]>('/nodes').then((r) => r.data),
    refetchInterval,
  })
}

export function useNode(nodeId: string) {
  return useQuery({
    queryKey: ['nodes', nodeId],
    queryFn: () => apiClient.get<Node>(`/nodes/${nodeId}`).then((r) => r.data),
    enabled: !!nodeId,
  })
}
