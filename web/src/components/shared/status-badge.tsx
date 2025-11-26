import { cn } from '@/lib/utils'

type Status =
  | 'online' | 'offline'
  | 'pending' | 'scheduled' | 'running' | 'completed' | 'failed' | 'succeeded' | 'unknown'
  | 'waiting' | 'terminated'

interface StatusBadgeProps {
  status: Status
  className?: string
}

const statusConfig: Record<Status, { label: string; color: string }> = {
  online: { label: 'Online', color: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' },
  offline: { label: 'Offline', color: 'bg-red-500/15 text-red-600 dark:text-red-400' },
  pending: { label: 'Pending', color: 'bg-amber-500/15 text-amber-600 dark:text-amber-400' },
  scheduled: { label: 'Scheduled', color: 'bg-blue-500/15 text-blue-600 dark:text-blue-400' },
  running: { label: 'Running', color: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' },
  completed: { label: 'Completed', color: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' },
  succeeded: { label: 'Succeeded', color: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400' },
  failed: { label: 'Failed', color: 'bg-red-500/15 text-red-600 dark:text-red-400' },
  unknown: { label: 'Unknown', color: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400' },
  waiting: { label: 'Waiting', color: 'bg-amber-500/15 text-amber-600 dark:text-amber-400' },
  terminated: { label: 'Terminated', color: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400' },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] || { label: status, color: 'bg-zinc-500/15 text-zinc-600' }

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
        config.color,
        className
      )}
    >
      {config.label}
    </span>
  )
}
