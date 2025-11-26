import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { useTasks } from '@/hooks'
import { StatusBadge, HealthBadge, TimestampDisplay } from '@/components/shared'

export function Tasks() {
  const { data: tasks, isLoading } = useTasks()

  return (
    <>
      <Header title="Tasks" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Tasks</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : tasks && tasks.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Image</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Health</TableHead>
                    <TableHead>Node</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tasks.map((task) => (
                    <TableRow key={task.taskId}>
                      <TableCell>
                        <Link
                          to={`/tasks/${task.taskId}`}
                          className="font-medium hover:underline"
                        >
                          {task.name}
                        </Link>
                      </TableCell>
                      <TableCell className="font-mono text-sm">{task.image}</TableCell>
                      <TableCell>
                        <StatusBadge status={task.status} />
                      </TableCell>
                      <TableCell>
                        <HealthBadge health={task.healthStatus} />
                      </TableCell>
                      <TableCell>
                        {task.nodeId ? (
                          <Link
                            to={`/nodes/${task.nodeId}`}
                            className="hover:underline"
                          >
                            {task.nodeId.slice(0, 12)}...
                          </Link>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <TimestampDisplay timestamp={task.createdAt} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No tasks</p>
            )}
          </CardContent>
        </Card>
      </main>
    </>
  )
}
