import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Container, Pod } from '@/api/types'
import { toast } from 'sonner'

export interface CreatePodRequest {
  name: string
  namespace?: string
  labels?: Record<string, string>
  containers: Omit<Container, 'containerId' | 'status' | 'healthStatus'>[]
}

export function useCreatePod() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreatePodRequest) =>
      apiClient.post<Pod>('/pods', data).then((r) => r.data),
    onSuccess: (pod) => {
      queryClient.invalidateQueries({ queryKey: ['pods'] })
      toast.success(`Pod "${pod.name}" created`)
    },
    onError: (error: Error) => {
      toast.error(`Failed to create pod: ${error.message}`)
    },
  })
}

export function useDeletePod() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (podId: string) => apiClient.delete(`/pods/${podId}`),
    onSuccess: (_, podId) => {
      queryClient.invalidateQueries({ queryKey: ['pods'] })
      queryClient.removeQueries({ queryKey: ['pods', podId] })
      toast.success('Pod deleted')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete pod: ${error.message}`)
    },
  })
}
