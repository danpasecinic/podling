import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/api/client'
import type { Service, Endpoints } from '@/api/types'

export function useServices(refetchInterval = 5000) {
  return useQuery({
    queryKey: ['services'],
    queryFn: () => apiClient.get<Service[]>('/services').then((r) => r.data),
    refetchInterval,
  })
}

export function useService(serviceId: string) {
  return useQuery({
    queryKey: ['services', serviceId],
    queryFn: () => apiClient.get<Service>(`/services/${serviceId}`).then((r) => r.data),
    enabled: !!serviceId,
  })
}

export function useEndpoints(serviceId: string) {
  return useQuery({
    queryKey: ['services', serviceId, 'endpoints'],
    queryFn: () => apiClient.get<Endpoints>(`/services/${serviceId}/endpoints`).then((r) => r.data),
    enabled: !!serviceId,
  })
}
