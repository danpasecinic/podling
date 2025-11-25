import { useParams, Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useNodes } from '@/hooks'
import { StatusBadge, ResourceGauge, TimestampDisplay } from '@/components/shared'
import { formatBytes, formatCPU } from '@/lib/formatters'
import { ArrowLeft } from 'lucide-react'

export function NodeDetail() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const { data: nodes, isLoading } = useNodes()

  const node = nodes?.find((n) => n.nodeId === nodeId)

  if (isLoading) {
    return (
      <>
        <Header title="Node Details" />
        <main className="flex-1 p-6">
          <Skeleton className="h-64 w-full" />
        </main>
      </>
    )
  }

  if (!node) {
    return (
      <>
        <Header title="Node Details" />
        <main className="flex-1 p-6">
          <Card>
            <CardContent className="pt-6">
              <p className="text-muted-foreground">Node not found</p>
              <Button asChild className="mt-4">
                <Link to="/nodes">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to Nodes
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
      <Header title={`Node: ${node.hostname}`} />
      <main className="flex-1 p-6">
        <Button asChild variant="ghost" className="mb-4">
          <Link to="/nodes">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Nodes
          </Link>
        </Button>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Status</span>
                <StatusBadge status={node.status} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Address</span>
                <span>{node.hostname}:{node.port}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Node ID</span>
                <span className="font-mono text-sm">{node.nodeId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Running Tasks</span>
                <span>{node.runningTasks}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Last Heartbeat</span>
                <TimestampDisplay timestamp={node.lastHeartbeat} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Resources</CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              {node.resources ? (
                <>
                  <ResourceGauge
                    label="CPU"
                    used={node.resources.used.cpu}
                    total={node.resources.capacity.cpu}
                    formatValue={formatCPU}
                  />
                  <ResourceGauge
                    label="Memory"
                    used={node.resources.used.memory}
                    total={node.resources.capacity.memory}
                    formatValue={formatBytes}
                  />
                </>
              ) : (
                <p className="text-sm text-muted-foreground">No resource information available</p>
              )}
            </CardContent>
          </Card>
        </div>
      </main>
    </>
  )
}
