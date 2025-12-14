import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Task } from '@/api/types'
import { toast } from 'sonner'

export interface CreateTaskRequest {
  name: string
  image: string
  env?: Record<string, string>
}

export function useCreateTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTaskRequest) =>
      apiClient.post<Task>('/tasks', data).then((r) => r.data),
    onSuccess: (task) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      toast.success(`Task "${task.name}" created`)
    },
    onError: (error: Error) => {
      toast.error(`Failed to create task: ${error.message}`)
    },
  })
}

export function useDeleteTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (taskId: string) => apiClient.delete(`/tasks/${taskId}`),
    onSuccess: (_, taskId) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] })
      queryClient.removeQueries({ queryKey: ['tasks', taskId] })
      toast.success('Task deleted')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete task: ${error.message}`)
    },
  })
}
