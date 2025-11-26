import { useParams, Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useService, useEndpoints } from '@/hooks'
import { LabelBadges, TimestampDisplay } from '@/components/shared'
import { ArrowLeft } from 'lucide-react'

export function ServiceDetail() {
  const { serviceId } = useParams<{ serviceId: string }>()
  const { data: service, isLoading } = useService(serviceId || '')
  const { data: endpoints } = useEndpoints(serviceId || '')

  if (isLoading) {
    return (
      <>
        <Header title="Service Details" />
        <main className="flex-1 p-6">
          <Skeleton className="h-64 w-full" />
        </main>
      </>
    )
  }

  if (!service) {
    return (
      <>
        <Header title="Service Details" />
        <main className="flex-1 p-6">
          <Card>
            <CardContent className="pt-6">
              <p className="text-muted-foreground">Service not found</p>
              <Button asChild className="mt-4">
                <Link to="/services">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to Services
                </Link>
              </Button>
            </CardContent>
          </Card>
        </main>
      </>
    )
  }

  const readyAddresses = endpoints?.subsets?.flatMap((s) => s.addresses) ?? []
  const notReadyAddresses = endpoints?.subsets?.flatMap((s) => s.notReadyAddresses ?? []) ?? []

  return (
    <>
      <Header title={`Service: ${service.name}`} />
      <main className="flex-1 p-6">
        <Button asChild variant="ghost" className="mb-4">
          <Link to="/services">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Services
          </Link>
        </Button>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Type</span>
                <Badge variant="outline">{service.type}</Badge>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Namespace</span>
                <span>{service.namespace || 'default'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Cluster IP</span>
                <span className="font-mono">{service.clusterIp || '-'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Service ID</span>
                <span className="font-mono text-sm">{service.serviceId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <TimestampDisplay timestamp={service.createdAt} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Selector</CardTitle>
            </CardHeader>
            <CardContent>
              <LabelBadges labels={service.selector} maxVisible={10} />
            </CardContent>
          </Card>
        </div>

        <Card className="mt-4">
          <CardHeader>
            <CardTitle>Ports</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Protocol</TableHead>
                  <TableHead>Port</TableHead>
                  <TableHead>Target Port</TableHead>
                  <TableHead>Node Port</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {service.ports.map((port, idx) => (
                  <TableRow key={idx}>
                    <TableCell>{port.name || '-'}</TableCell>
                    <TableCell>{port.protocol || 'TCP'}</TableCell>
                    <TableCell>{port.port}</TableCell>
                    <TableCell>{port.targetPort || port.port}</TableCell>
                    <TableCell>{port.nodePort || '-'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card className="mt-4">
          <CardHeader>
            <CardTitle>
              Endpoints ({readyAddresses.length} ready, {notReadyAddresses.length} not ready)
            </CardTitle>
          </CardHeader>
          <CardContent>
            {readyAddresses.length > 0 || notReadyAddresses.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IP</TableHead>
                    <TableHead>Pod</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {readyAddresses.map((addr, idx) => (
                    <TableRow key={`ready-${idx}`}>
                      <TableCell className="font-mono">{addr.ip}</TableCell>
                      <TableCell>
                        <Link to={`/pods/${addr.podId}`} className="hover:underline">
                          {addr.podId.slice(0, 16)}...
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge variant="default">Ready</Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                  {notReadyAddresses.map((addr, idx) => (
                    <TableRow key={`notready-${idx}`}>
                      <TableCell className="font-mono">{addr.ip}</TableCell>
                      <TableCell>
                        <Link to={`/pods/${addr.podId}`} className="hover:underline">
                          {addr.podId.slice(0, 16)}...
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">Not Ready</Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No endpoints</p>
            )}
          </CardContent>
        </Card>
      </main>
    </>
  )
}
