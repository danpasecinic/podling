import { Badge } from '@/components/ui/badge'
import { Heart, HeartOff, HelpCircle } from 'lucide-react'
import type { HealthStatus } from '@/api/types'

interface HealthBadgeProps {
  health: HealthStatus
  showLabel?: boolean
}

export function HealthBadge({ health, showLabel = true }: HealthBadgeProps) {
  const config = {
    healthy: { icon: Heart, label: 'Healthy', variant: 'default' as const, className: 'text-green-500' },
    unhealthy: { icon: HeartOff, label: 'Unhealthy', variant: 'destructive' as const, className: 'text-red-500' },
    unknown: { icon: HelpCircle, label: 'Unknown', variant: 'outline' as const, className: 'text-muted-foreground' },
  }

  const { icon: Icon, label, variant, className } = config[health]

  return (
    <Badge variant={variant} className="gap-1">
      <Icon className={`h-3 w-3 ${className}`} />
      {showLabel && label}
    </Badge>
  )
}
