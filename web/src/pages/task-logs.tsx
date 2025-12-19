import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Header } from '@/components/layout'
import { LogViewer } from '@/components/shared'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useTask, useTaskLogStream } from '@/hooks'
import { ArrowLeft, Radio, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'

export function TaskLogs() {
  const { taskId } = useParams<{ taskId: string }>()
  const { data: task, isLoading } = useTask(taskId || '')

  const {
    logs,
    isConnected,
    isStreaming,
    connect,
    toggleStream,
    clearLogs,
  } = useTaskLogStream(taskId || '', {
    onError: (error) => toast.error(`Log stream error: ${error}`),
  })

  useEffect(() => {
    if (task && task.status === 'running') {
      connect()
    }
  }, [task, connect])

  if (isLoading) {
    return (
      <>
        <Header title="Task Logs" />
        <main className="flex-1 p-6">
          <Skeleton className="h-[600px] w-full" />
        </main>
      </>
    )
  }

  if (!task) {
    return (
      <>
        <Header title="Task Logs" />
        <main className="flex-1 p-6">
          <Card>
            <CardContent className="pt-6">
              <p className="text-muted-foreground">Task not found</p>
              <Button asChild className="mt-4">
                <Link to="/tasks">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to Tasks
                </Link>
              </Button>
            </CardContent>
          </Card>
        </main>
      </>
    )
  }

  const canStream = task.status === 'running'

  return (
    <>
      <Header title={`Logs: ${task.name}`} />
      <main className="flex-1 p-6 flex flex-col min-h-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-4">
            <Button asChild variant="ghost">
              <Link to={`/tasks/${taskId}`}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Task
              </Link>
            </Button>
            <span className="text-sm text-muted-foreground font-mono">{task.taskId}</span>
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

        <Card className="flex-1 min-h-0 flex flex-col">
          <CardHeader className="py-3 px-4 border-b">
            <CardTitle className="text-sm font-medium">Container Output</CardTitle>
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
