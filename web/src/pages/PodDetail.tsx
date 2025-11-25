import { useParams, Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { usePod } from '@/api'
import { StatusBadge, HealthBadge, LabelBadges, TimestampDisplay } from '@/components/shared'
import { ArrowLeft } from 'lucide-react'

export function PodDetail() {
  const { podId } = useParams<{ podId: string }>()
  const { data: pod, isLoading } = usePod(podId || '')

  if (isLoading) {
    return (
      <>
        <Header title="Pod Details" />
        <main className="flex-1 p-6">
          <Skeleton className="h-64 w-full" />
        </main>
      </>
    )
  }

  if (!pod) {
    return (
      <>
        <Header title="Pod Details" />
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

  return (
    <>
      <Header title={`Pod: ${pod.name}`} />
      <main className="flex-1 p-6">
        <Button asChild variant="ghost" className="mb-4">
          <Link to="/pods">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Pods
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
                <StatusBadge status={pod.status} />
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Namespace</span>
                <span>{pod.namespace || 'default'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Pod ID</span>
                <span className="font-mono text-sm">{pod.podId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Node</span>
                {pod.nodeId ? (
                  <Link to={`/nodes/${pod.nodeId}`} className="hover:underline">
                    {pod.nodeId.slice(0, 16)}...
                  </Link>
                ) : (
                  <span>-</span>
                )}
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Restart Policy</span>
                <span>{pod.restartPolicy || 'Never'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <TimestampDisplay timestamp={pod.createdAt} />
              </div>
              {pod.message && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Message</span>
                  <span>{pod.message}</span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Labels & Annotations</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <span className="text-sm text-muted-foreground">Labels</span>
                <div className="mt-1">
                  <LabelBadges labels={pod.labels} maxVisible={10} />
                </div>
              </div>
              <div>
                <span className="text-sm text-muted-foreground">Annotations</span>
                <div className="mt-1">
                  <LabelBadges labels={pod.annotations} maxVisible={10} />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card className="mt-4">
          <CardHeader>
            <CardTitle>Containers ({pod.containers.length})</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Image</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Health</TableHead>
                  <TableHead>Restarts</TableHead>
                  <TableHead>Started</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pod.containers.map((container) => (
                  <TableRow key={container.name}>
                    <TableCell className="font-medium">{container.name}</TableCell>
                    <TableCell className="font-mono text-sm">{container.image}</TableCell>
                    <TableCell>
                      <StatusBadge status={container.status} />
                    </TableCell>
                    <TableCell>
                      <HealthBadge health={container.healthStatus} />
                    </TableCell>
                    <TableCell>{container.restartCount ?? 0}</TableCell>
                    <TableCell>
                      <TimestampDisplay timestamp={container.startedAt} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </main>
    </>
  )
}
