import { Badge } from '@/components/ui/badge'

interface LabelBadgesProps {
  labels?: Record<string, string>
  maxVisible?: number
}

export function LabelBadges({ labels, maxVisible = 3 }: LabelBadgesProps) {
  if (!labels || Object.keys(labels).length === 0) {
    return <span className="text-muted-foreground text-sm">-</span>
  }

  const entries = Object.entries(labels)
  const visible = entries.slice(0, maxVisible)
  const remaining = entries.length - maxVisible

  return (
    <div className="flex flex-wrap gap-1">
      {visible.map(([key, value]) => (
        <Badge key={key} variant="outline" className="text-xs font-normal">
          {key}={value}
        </Badge>
      ))}
      {remaining > 0 && (
        <Badge variant="secondary" className="text-xs">
          +{remaining} more
        </Badge>
      )}
    </div>
  )
}
