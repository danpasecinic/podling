import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type Status =
  | 'online' | 'offline'
  | 'pending' | 'scheduled' | 'running' | 'completed' | 'failed' | 'succeeded' | 'unknown'
  | 'waiting' | 'terminated'

interface StatusBadgeProps {
  status: Status
  className?: string
}

const statusConfig: Record<Status, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  online: { label: 'Online', variant: 'default' },
  offline: { label: 'Offline', variant: 'destructive' },
  pending: { label: 'Pending', variant: 'secondary' },
  scheduled: { label: 'Scheduled', variant: 'secondary' },
  running: { label: 'Running', variant: 'default' },
  completed: { label: 'Completed', variant: 'default' },
  succeeded: { label: 'Succeeded', variant: 'default' },
  failed: { label: 'Failed', variant: 'destructive' },
  unknown: { label: 'Unknown', variant: 'outline' },
  waiting: { label: 'Waiting', variant: 'secondary' },
  terminated: { label: 'Terminated', variant: 'outline' },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] || { label: status, variant: 'outline' as const }

  return (
    <Badge variant={config.variant} className={cn('capitalize', className)}>
      {config.label}
    </Badge>
  )
}
