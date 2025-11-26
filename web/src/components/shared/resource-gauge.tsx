import { cn } from '@/lib/utils'

interface ResourceGaugeProps {
  label: string
  used: number
  total: number
  formatValue: (value: number) => string
  className?: string
}

export function ResourceGauge({ label, used, total, formatValue, className }: ResourceGaugeProps) {
  const safeUsed = used ?? 0
  const safeTotal = total ?? 0
  const percentage = safeTotal > 0 ? Math.round((safeUsed / safeTotal) * 100) : 0
  const getColor = () => {
    if (percentage >= 90) return 'bg-red-500'
    if (percentage >= 70) return 'bg-yellow-500'
    return 'bg-green-500'
  }

  return (
    <div className={cn('space-y-1', className)}>
      <div className="flex justify-between text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium">
          {formatValue(safeUsed)} / {formatValue(safeTotal)}
        </span>
      </div>
      <div className="h-2 bg-secondary rounded-full overflow-hidden">
        <div
          className={cn('h-full transition-all duration-300', getColor())}
          style={{ width: `${percentage}%` }}
        />
      </div>
      <div className="text-xs text-muted-foreground text-right">{percentage}%</div>
    </div>
  )
}
