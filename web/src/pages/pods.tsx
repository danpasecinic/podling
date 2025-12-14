import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { CreatePodDialog } from '@/components/forms'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { usePods, useTableSort } from '@/hooks'
import { LabelBadges, SortableHeader, StatusBadge, TimestampDisplay } from '@/components/shared'
import type { Pod } from '@/api/types'

export function Pods() {
  const { data: pods, isLoading } = usePods()
  const { sortedData, sort, toggleSort } = useTableSort<Pod>(pods, 'name')

  return (
    <>
      <Header title="Pods" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Pods</CardTitle>
            <CreatePodDialog />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : sortedData && sortedData.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>
                      <SortableHeader label="Name" sortKey="name" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>
                      <SortableHeader label="Namespace" sortKey="namespace" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>
                      <SortableHeader label="Status" sortKey="status" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>Containers</TableHead>
                    <TableHead>
                      <SortableHeader label="Node" sortKey="nodeId" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>Labels</TableHead>
                    <TableHead>
                      <SortableHeader label="Age" sortKey="createdAt" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedData.map((pod) => {
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
