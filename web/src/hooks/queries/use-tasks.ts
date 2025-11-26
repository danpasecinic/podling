import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Task } from '@/api/types'

export function useTasks(refetchInterval = 5000) {
  return useQuery({
    queryKey: ['tasks'],
    queryFn: () => apiClient.get<Task[]>('/tasks').then((r) => r.data),
    refetchInterval,
  })
}

export function useTask(taskId: string) {
  return useQuery({
    queryKey: ['tasks', taskId],
    queryFn: () => apiClient.get<Task>(`/tasks/${taskId}`).then((r) => r.data),
    enabled: !!taskId,
  })
}
