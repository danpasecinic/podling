import { Link, useNavigate, useParams } from 'react-router-dom'
import { Header } from '@/components/layout'
import { DeleteConfirmationDialog } from '@/components/forms'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useDeleteTask, useTask } from '@/hooks'
import { HealthBadge, StatusBadge, TimestampDisplay } from '@/components/shared'
import { ArrowLeft, Trash2, ScrollText } from 'lucide-react'

export function TaskDetail() {
  const { taskId } = useParams<{ taskId: string }>()
  const navigate = useNavigate()
  const { data: task, isLoading } = useTask(taskId || '')
  const deleteTask = useDeleteTask()

  const handleDelete = () => {
    if (taskId) {
      deleteTask.mutate(taskId, {
        onSuccess: () => navigate('/tasks'),
      })
    }
  }

  if (isLoading) {
    return (
      <>
        <Header title="Task Details" />
        <main className="flex-1 p-6">
          <Skeleton className="h-64 w-full" />
        </main>
      </>
    )
  }

  if (!task) {
    return (
      <>
        <Header title="Task Details" />
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

  return (
    <>
      <Header title={`Task: ${task.name}`} />
      <main className="flex-1 p-6">
        <div className="flex items-center justify-between mb-4">
          <Button asChild variant="ghost">
            <Link to="/tasks">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Tasks
            </Link>
          </Button>
          <div className="flex items-center gap-2">
            <Button asChild variant="outline" size="sm">
              <Link to={`/tasks/${taskId}/logs`}>
                <ScrollText className="mr-2 h-4 w-4" />
                View Logs
              </Link>
            </Button>
            <DeleteConfirmationDialog
              title="Delete Task"
              description={`Are you sure you want to delete task "${task.name}"? This action cannot be undone.`}
              onConfirm={handleDelete}
              isPending={deleteTask.isPending}
              trigger={
                <Button variant="destructive" size="sm">
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Task
                </Button>
              }
            />
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Status</span>
                <StatusBadge status={task.status} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Health</span>
                <HealthBadge health={task.healthStatus} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Image</span>
                <span className="font-mono text-sm">{task.image}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Task ID</span>
                <span className="font-mono text-sm">{task.taskId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Node</span>
                {task.nodeId ? (
                  <Link to={`/nodes/${task.nodeId}`} className="hover:underline">
                    {task.nodeId.slice(0, 16)}...
                  </Link>
                ) : (
                  <span>-</span>
                )}
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Container ID</span>
                <span className="font-mono text-sm">
                  {task.containerId ? `${task.containerId.slice(0, 12)}...` : '-'}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Restart Policy</span>
                <span>{task.restartPolicy || 'Never'}</span>
              </div>
              {task.error && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Error</span>
                  <span className="text-destructive">{task.error}</span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Timestamps</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <TimestampDisplay timestamp={task.createdAt} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Started</span>
                <TimestampDisplay timestamp={task.startedAt} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Finished</span>
                <TimestampDisplay timestamp={task.finishedAt} />
              </div>
            </CardContent>
          </Card>
        </div>

        {task.env && Object.keys(task.env).length > 0 && (
          <Card className="mt-4">
            <CardHeader>
              <CardTitle>Environment Variables</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {Object.entries(task.env).map(([key, value]) => (
                  <Badge key={key} variant="outline" className="font-mono text-xs">
                    {key}={value}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {task.ports && task.ports.length > 0 && (
          <Card className="mt-4">
            <CardHeader>
              <CardTitle>Ports</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {task.ports.map((port, idx) => (
                  <Badge key={idx} variant="secondary">
                    {port.name ? `${port.name}: ` : ''}
                    {port.containerPort}
                    {port.hostPort ? ` -> ${port.hostPort}` : ''}
                    /{port.protocol || 'TCP'}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </main>
    </>
  )
}
