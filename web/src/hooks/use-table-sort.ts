import { useState, useMemo } from 'react'

export type SortDirection = 'asc' | 'desc'

export interface SortState<T> {
  key: keyof T | null
  direction: SortDirection
}

export function useTableSort<T>(data: T[] | undefined, defaultKey?: keyof T) {
  const [sort, setSort] = useState<SortState<T>>({
    key: defaultKey ?? null,
    direction: 'asc',
  })

  const toggleSort = (key: keyof T) => {
    setSort((prev) => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc',
    }))
  }

  const sortedData = useMemo(() => {
    if (!data || !sort.key) return data

    return [...data].sort((a, b) => {
      const aVal = a[sort.key!]
      const bVal = b[sort.key!]

      if (aVal === null || aVal === undefined) return 1
      if (bVal === null || bVal === undefined) return -1

      let comparison: number
      if (typeof aVal === 'string' && typeof bVal === 'string') {
        comparison = aVal.localeCompare(bVal)
      } else if (typeof aVal === 'number' && typeof bVal === 'number') {
        comparison = aVal - bVal
      } else if (aVal instanceof Date && bVal instanceof Date) {
        comparison = aVal.getTime() - bVal.getTime()
      } else {
        comparison = String(aVal).localeCompare(String(bVal))
      }

      return sort.direction === 'asc' ? comparison : -comparison
    })
  }, [data, sort])

  return { sortedData, sort, toggleSort }
}
