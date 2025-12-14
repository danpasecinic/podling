import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Service, ServicePort } from '@/api/types'
import { toast } from 'sonner'

export interface CreateServiceRequest {
  name: string
  namespace?: string
  selector?: Record<string, string>
  ports: ServicePort[]
  labels?: Record<string, string>
}

export function useCreateService() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateServiceRequest) =>
      apiClient.post<Service>('/services', data).then((r) => r.data),
    onSuccess: (service) => {
      queryClient.invalidateQueries({ queryKey: ['services'] })
      toast.success(`Service "${service.name}" created`)
    },
    onError: (error: Error) => {
      toast.error(`Failed to create service: ${error.message}`)
    },
  })
}

export function useDeleteService() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (serviceId: string) => apiClient.delete(`/services/${serviceId}`),
    onSuccess: (_, serviceId) => {
      queryClient.invalidateQueries({ queryKey: ['services'] })
      queryClient.removeQueries({ queryKey: ['services', serviceId] })
      toast.success('Service deleted')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete service: ${error.message}`)
    },
  })
}
