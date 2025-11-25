import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useServices } from '@/api'
import { LabelBadges, TimestampDisplay } from '@/components/shared'

export function Services() {
  const { data: services, isLoading } = useServices()

  return (
    <>
      <Header title="Services" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Services</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : services && services.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Namespace</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Cluster IP</TableHead>
                    <TableHead>Ports</TableHead>
                    <TableHead>Selector</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {services.map((service) => (
                    <TableRow key={service.serviceId}>
                      <TableCell>
                        <Link
                          to={`/services/${service.serviceId}`}
                          className="font-medium hover:underline"
                        >
                          {service.name}
                        </Link>
                      </TableCell>
                      <TableCell>{service.namespace || 'default'}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{service.type}</Badge>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {service.clusterIp || '-'}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {service.ports.slice(0, 2).map((port, idx) => (
                            <Badge key={idx} variant="secondary" className="text-xs">
                              {port.name ? `${port.name}: ` : ''}
                              {port.port}
                              {port.targetPort ? ` -> ${port.targetPort}` : ''}
                            </Badge>
                          ))}
                          {service.ports.length > 2 && (
                            <Badge variant="secondary" className="text-xs">
                              +{service.ports.length - 2}
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <LabelBadges labels={service.selector} maxVisible={2} />
                      </TableCell>
                      <TableCell>
                        <TimestampDisplay timestamp={service.createdAt} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">No services</p>
            )}
          </CardContent>
        </Card>
      </main>
    </>
  )
}
