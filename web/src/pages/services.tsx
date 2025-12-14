import { Link } from 'react-router-dom'
import { Header } from '@/components/layout'
import { CreateServiceDialog } from '@/components/forms'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useServices, useTableSort } from '@/hooks'
import { LabelBadges, SortableHeader, TimestampDisplay } from '@/components/shared'
import type { Service } from '@/api/types'

export function Services() {
  const { data: services, isLoading } = useServices()
  const { sortedData, sort, toggleSort } = useTableSort<Service>(services, 'name')

  return (
    <>
      <Header title="Services" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Services</CardTitle>
            <CreateServiceDialog />
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
                      <SortableHeader label="Type" sortKey="type" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>
                      <SortableHeader label="Cluster IP" sortKey="clusterIp" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                    <TableHead>Ports</TableHead>
                    <TableHead>Selector</TableHead>
                    <TableHead>
                      <SortableHeader label="Created" sortKey="createdAt" currentSort={sort} onSort={toggleSort} />
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedData.map((service) => (
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
