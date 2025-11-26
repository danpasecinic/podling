import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { SortDirection } from '@/hooks/use-table-sort'

interface SortableHeaderProps<T> {
  label: string
  sortKey: keyof T
  currentSort: { key: keyof T | null; direction: SortDirection }
  onSort: (key: keyof T) => void
  className?: string
}

export function SortableHeader<T>({
  label,
  sortKey,
  currentSort,
  onSort,
  className,
}: SortableHeaderProps<T>) {
  const isActive = currentSort.key === sortKey

  return (
    <button
      onClick={() => onSort(sortKey)}
      className={cn(
        'inline-flex items-center gap-1 hover:text-foreground transition-colors -ml-2 px-2 py-1 rounded',
        isActive ? 'text-foreground' : 'text-muted-foreground',
        className
      )}
    >
      {label}
      {isActive ? (
        currentSort.direction === 'asc' ? (
          <ChevronUp className="h-4 w-4" />
        ) : (
          <ChevronDown className="h-4 w-4" />
        )
      ) : (
        <ChevronsUpDown className="h-3.5 w-3.5 opacity-50" />
      )}
    </button>
  )
}
