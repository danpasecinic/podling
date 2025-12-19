import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { Header } from '@/components/layout'
import { LogViewer } from '@/components/shared'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { usePod, usePodLogStream } from '@/hooks'
import { ArrowLeft, Radio, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'

export function PodLogs() {
  const { podId } = useParams<{ podId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedContainer = searchParams.get('container') || ''
  const { data: pod, isLoading } = usePod(podId || '')

  const [activeContainer, setActiveContainer] = useState(selectedContainer)

  useEffect(() => {
    if (pod && pod.containers.length > 0 && !activeContainer) {
      setActiveContainer(pod.containers[0].name)
    }
  }, [pod, activeContainer])

  const {
    logs,
    isConnected,
    isStreaming,
    connect,
    disconnect,
    toggleStream,
    clearLogs,
  } = usePodLogStream(podId || '', activeContainer, {
    onError: (error) => toast.error(`Log stream error: ${error}`),
  })

  useEffect(() => {
    if (pod && pod.status === 'running' && activeContainer) {
      disconnect()
      clearLogs()
      connect()
    }
    return () => {
      disconnect()
    }
  }, [activeContainer])

  const handleContainerChange = (container: string) => {
    setActiveContainer(container)
    setSearchParams(container ? { container } : {})
  }

  if (isLoading) {
    return (
      <>
        <Header title="Pod Logs" />
        <main className="flex-1 p-6">
          <Skeleton className="h-[600px] w-full" />
        </main>
      </>
    )
  }

  if (!pod) {
    return (
      <>
        <Header title="Pod Logs" />
        <main className="flex-1 p-6">
          <Card>
            <CardContent className="pt-6">
              <p className="text-muted-foreground">Pod not found</p>
              <Button asChild className="mt-4">
                <Link to="/pods">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to Pods
                </Link>
              </Button>
            </CardContent>
          </Card>
        </main>
      </>
    )
  }

  const canStream = pod.status === 'running'

  return (
    <>
      <Header title={`Logs: ${pod.name}`} />
      <main className="flex-1 p-6 flex flex-col min-h-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-4">
            <Button asChild variant="ghost">
              <Link to={`/pods/${podId}`}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Pod
              </Link>
            </Button>
            <span className="text-sm text-muted-foreground font-mono">{pod.podId}</span>
          </div>
          <div className="flex items-center gap-2">
            {canStream && !isConnected && (
              <Button variant="outline" size="sm" onClick={connect}>
                <Radio className="mr-2 h-4 w-4" />
                Start Streaming
              </Button>
            )}
            <Button variant="outline" size="sm" onClick={() => { clearLogs(); connect() }}>
              <RotateCcw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>

        {pod.containers.length > 1 && (
          <Tabs value={activeContainer} onValueChange={handleContainerChange} className="mb-4">
            <TabsList>
              {pod.containers.map((container) => (
                <TabsTrigger key={container.name} value={container.name}>
                  {container.name}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        )}

        <Card className="flex-1 min-h-0 flex flex-col">
          <CardHeader className="py-3 px-4 border-b">
            <CardTitle className="text-sm font-medium">
              Container: {activeContainer || 'Select a container'}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 p-0 min-h-0">
            <LogViewer
              logs={logs}
              isStreaming={isStreaming}
              isConnected={isConnected}
              onToggleStream={canStream ? toggleStream : undefined}
              onClear={clearLogs}
              className="h-full"
            />
          </CardContent>
        </Card>
      </main>
    </>
  )
}
