import { useQuery } from '@tanstack/react-query'
import { useCallback, useEffect, useRef, useState } from 'react'
import { apiClient } from '@/api/client'
import { getStoredAuth } from '@/api/auth'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

interface TaskLogsResponse {
  taskId: string
  logs: string
  tail: number
}

interface PodLogsResponse {
  podId: string
  logs: Record<string, string>
  tail: number
}

export function useTaskLogs(taskId: string, tail = 100, enabled = true) {
  return useQuery({
    queryKey: ['taskLogs', taskId, tail],
    queryFn: () =>
      apiClient.get<TaskLogsResponse>(`/tasks/${taskId}/logs?tail=${tail}`).then((r) => r.data),
    enabled: enabled && !!taskId,
    refetchInterval: 5000,
  })
}

export function usePodLogs(podId: string, container?: string, tail = 100, enabled = true) {
  const containerParam = container ? `&container=${container}` : ''
  return useQuery({
    queryKey: ['podLogs', podId, container, tail],
    queryFn: () =>
      apiClient
        .get<PodLogsResponse>(`/pods/${podId}/logs?tail=${tail}${containerParam}`)
        .then((r) => r.data),
    enabled: enabled && !!podId,
    refetchInterval: 5000,
  })
}

interface UseLogStreamOptions {
  tail?: number
  onError?: (error: string) => void
}

export function useTaskLogStream(taskId: string, options: UseLogStreamOptions = {}) {
  const { tail = 100, onError } = options
  const [logs, setLogs] = useState<string[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)

  const connect = useCallback(() => {
    if (!taskId) return

    const auth = getStoredAuth()
    let url = `${API_BASE_URL}/api/v1/tasks/${taskId}/logs/stream?tail=${tail}`
    if (auth?.token) {
      url += `&token=${auth.token}`
    }

    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setIsConnected(true)
      setIsStreaming(true)
    }

    eventSource.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      setIsStreaming(false)
      eventSource.close()
      onError?.('Connection lost')
    }

    eventSource.addEventListener('error', (event) => {
      const messageEvent = event as MessageEvent
      if (messageEvent.data) {
        onError?.(messageEvent.data)
      }
    })
  }, [taskId, tail, onError])

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
      setIsConnected(false)
      setIsStreaming(false)
    }
  }, [])

  const toggleStream = useCallback(() => {
    if (isStreaming) {
      disconnect()
    } else {
      connect()
    }
  }, [isStreaming, connect, disconnect])

  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  return {
    logs,
    isConnected,
    isStreaming,
    connect,
    disconnect,
    toggleStream,
    clearLogs,
  }
}

export function usePodLogStream(
  podId: string,
  container?: string,
  options: UseLogStreamOptions = {}
) {
  const { tail = 100, onError } = options
  const [logs, setLogs] = useState<string[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)

  const connect = useCallback(() => {
    if (!podId) return

    const auth = getStoredAuth()
    let url = `${API_BASE_URL}/api/v1/pods/${podId}/logs/stream?tail=${tail}`
    if (container) {
      url += `&container=${container}`
    }
    if (auth?.token) {
      url += `&token=${auth.token}`
    }

    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setIsConnected(true)
      setIsStreaming(true)
    }

    eventSource.onmessage = (event) => {
      setLogs((prev) => [...prev, event.data])
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      setIsStreaming(false)
      eventSource.close()
      onError?.('Connection lost')
    }

    eventSource.addEventListener('error', (event) => {
      const messageEvent = event as MessageEvent
      if (messageEvent.data) {
        onError?.(messageEvent.data)
      }
    })
  }, [podId, container, tail, onError])

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
      setIsConnected(false)
      setIsStreaming(false)
    }
  }, [])

  const toggleStream = useCallback(() => {
    if (isStreaming) {
      disconnect()
    } else {
      connect()
    }
  }, [isStreaming, connect, disconnect])

  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  return {
    logs,
    isConnected,
    isStreaming,
    connect,
    disconnect,
    toggleStream,
    clearLogs,
  }
}
