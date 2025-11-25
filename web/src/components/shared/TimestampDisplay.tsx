import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatRelativeTime, formatDateTime } from '@/lib/formatters'

interface TimestampDisplayProps {
  timestamp?: string
  fallback?: string
}

export function TimestampDisplay({ timestamp, fallback = '-' }: TimestampDisplayProps) {
  if (!timestamp) {
    return <span className="text-muted-foreground">{fallback}</span>
  }

  return (
    <Tooltip>
      <TooltipTrigger className="cursor-default">
        {formatRelativeTime(timestamp)}
      </TooltipTrigger>
      <TooltipContent>
        {formatDateTime(timestamp)}
      </TooltipContent>
    </Tooltip>
  )
}
