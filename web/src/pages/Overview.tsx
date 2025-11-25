import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useNodes, usePods, useTasks, useServices } from '@/api'
import { Server, Box, PlayCircle, Network } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'

function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
  isLoading,
}: {
  title: string
  value: number
  subtitle?: string
  icon: React.ElementType
  isLoading: boolean
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <>
            <div className="text-2xl font-bold">{value}</div>
            {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
          </>
        )}
      </CardContent>
    </Card>
  )
}

export function Overview() {
  const { data: nodes, isLoading: nodesLoading } = useNodes()
  const { data: pods, isLoading: podsLoading } = usePods()
  const { data: tasks, isLoading: tasksLoading } = useTasks()
  const { data: services, isLoading: servicesLoading } = useServices()

  const onlineNodes = nodes?.filter((n) => n.status === 'online').length ?? 0
  const totalNodes = nodes?.length ?? 0

  const runningPods = pods?.filter((p) => p.status === 'running').length ?? 0
  const totalPods = pods?.length ?? 0

  const runningTasks = tasks?.filter((t) => t.status === 'running').length ?? 0
  const totalTasks = tasks?.length ?? 0

  return (
    <>
      <Header title="Overview" />
      <main className="flex-1 p-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <StatCard
            title="Nodes"
            value={totalNodes}
            subtitle={`${onlineNodes} online`}
            icon={Server}
            isLoading={nodesLoading}
          />
          <StatCard
            title="Pods"
            value={totalPods}
            subtitle={`${runningPods} running`}
            icon={Box}
            isLoading={podsLoading}
          />
          <StatCard
            title="Tasks"
            value={totalTasks}
            subtitle={`${runningTasks} running`}
            icon={PlayCircle}
            isLoading={tasksLoading}
          />
          <StatCard
            title="Services"
            value={services?.length ?? 0}
            icon={Network}
            isLoading={servicesLoading}
          />
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Recent Pods</CardTitle>
            </CardHeader>
            <CardContent>
              {podsLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-full" />
                </div>
              ) : pods && pods.length > 0 ? (
                <div className="space-y-2">
                  {pods.slice(0, 5).map((pod) => (
                    <div key={pod.podId} className="flex items-center justify-between text-sm">
                      <span className="font-medium">{pod.name}</span>
                      <span className="text-muted-foreground capitalize">{pod.status}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No pods</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Recent Tasks</CardTitle>
            </CardHeader>
            <CardContent>
              {tasksLoading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-full" />
                </div>
              ) : tasks && tasks.length > 0 ? (
                <div className="space-y-2">
                  {tasks.slice(0, 5).map((task) => (
                    <div key={task.taskId} className="flex items-center justify-between text-sm">
                      <span className="font-medium">{task.name}</span>
                      <span className="text-muted-foreground capitalize">{task.status}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No tasks</p>
              )}
            </CardContent>
          </Card>
        </div>
      </main>
    </>
  )
}
