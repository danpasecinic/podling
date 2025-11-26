import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { useNodes } from '@/hooks'
import { StatusBadge, TimestampDisplay, ResourceGauge } from '@/components/shared'
import { formatBytes, formatCPU } from '@/lib/formatters'

export function Nodes() {
  const { data: nodes, isLoading } = useNodes()

  return (
    <>
      <Header title="Nodes" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Worker Nodes</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : nodes && nodes.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Hostname</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Running Tasks</TableHead>
                    <TableHead>CPU</TableHead>
                    <TableHead>Memory</TableHead>
                    <TableHead>Last Heartbeat</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {nodes.map((node) => (
                    <TableRow key={node.nodeId}>
                      <TableCell>
                        <Link
                          to={`/nodes/${node.nodeId}`}
                          className="font-medium hover:underline"
                        >
                          {node.hostname}:{node.port}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={node.status} />
                      </TableCell>
                      <TableCell>{node.runningTasks}</TableCell>
                      <TableCell>
                        {node.resources ? (
                          <ResourceGauge
                            label=""
                            used={node.resources.used.cpu}
                            total={node.resources.capacity.cpu}
                            formatValue={formatCPU}
                            className="w-32"
                          />
                        ) : (
                          '-'
                        )}
                      </TableCell>
                      <TableCell>
                        {node.resources ? (
                          <ResourceGauge
                            label=""
                            used={node.resources.used.memory}
                            total={node.resources.capacity.memory}
                            formatValue={formatBytes}
                            className="w-32"
                          />
                        ) : (
                          '-'
                        )}
                      </TableCell>
                      <TableCell>
                        <TimestampDisplay timestamp={node.lastHeartbeat} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No nodes registered</p>
            )}
          </CardContent>
        </Card>
      </main>
    </>
  )
}
