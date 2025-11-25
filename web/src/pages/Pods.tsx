import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { usePods } from '@/api'
import { StatusBadge, LabelBadges, TimestampDisplay } from '@/components/shared'

export function Pods() {
  const { data: pods, isLoading } = usePods()

  return (
    <>
      <Header title="Pods" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Pods</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : pods && pods.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Namespace</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Containers</TableHead>
                    <TableHead>Node</TableHead>
                    <TableHead>Labels</TableHead>
                    <TableHead>Age</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pods.map((pod) => {
                    const runningContainers = pod.containers.filter(
                      (c) => c.status === 'running'
                    ).length
                    return (
                      <TableRow key={pod.podId}>
                        <TableCell>
                          <Link
                            to={`/pods/${pod.podId}`}
                            className="font-medium hover:underline"
                          >
                            {pod.name}
                          </Link>
                        </TableCell>
                        <TableCell>{pod.namespace || 'default'}</TableCell>
                        <TableCell>
                          <StatusBadge status={pod.status} />
                        </TableCell>
                        <TableCell>
                          {runningContainers}/{pod.containers.length}
                        </TableCell>
                        <TableCell>
                          {pod.nodeId ? (
                            <Link
                              to={`/nodes/${pod.nodeId}`}
                              className="hover:underline"
                            >
                              {pod.nodeId.slice(0, 12)}...
                            </Link>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <LabelBadges labels={pod.labels} />
                        </TableCell>
                        <TableCell>
                          <TimestampDisplay timestamp={pod.createdAt} />
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No pods</p>
            )}
          </CardContent>
        </Card>
      </main>
    </>
  )
}
