import { useEffect, useRef, useState } from 'react'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Pause, Play, ArrowDown, Download, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface LogViewerProps {
  logs: string[]
  isStreaming?: boolean
  isConnected?: boolean
  onToggleStream?: () => void
  onClear?: () => void
  className?: string
}

export function LogViewer({
  logs,
  isStreaming = false,
  isConnected = false,
  onToggleStream,
  onClear,
  className,
}: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const [showScrollButton, setShowScrollButton] = useState(false)

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      const scrollContainer = scrollRef.current.querySelector('[data-radix-scroll-area-viewport]')
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight
      }
    }
  }, [logs, autoScroll])

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLDivElement
    const isAtBottom = target.scrollHeight - target.scrollTop - target.clientHeight < 50
    setAutoScroll(isAtBottom)
    setShowScrollButton(!isAtBottom && logs.length > 20)
  }

  const scrollToBottom = () => {
    if (scrollRef.current) {
      const scrollContainer = scrollRef.current.querySelector('[data-radix-scroll-area-viewport]')
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight
        setAutoScroll(true)
        setShowScrollButton(false)
      }
    }
  }

  const handleDownload = () => {
    const content = logs.join('\n')
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `logs-${new Date().toISOString()}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div className={cn('flex flex-col h-full', className)}>
      <div className="flex items-center justify-between gap-2 px-4 py-2 border-b bg-muted/50">
        <div className="flex items-center gap-2">
          <Badge variant={isConnected ? 'default' : 'secondary'} className="text-xs">
            {isConnected ? 'Connected' : 'Disconnected'}
          </Badge>
          {isStreaming && (
            <Badge variant="outline" className="text-xs">
              <span className="w-2 h-2 bg-green-500 rounded-full mr-1 animate-pulse" />
              Live
            </Badge>
          )}
          <span className="text-xs text-muted-foreground">{logs.length} lines</span>
        </div>
        <div className="flex items-center gap-1">
          {onToggleStream && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggleStream}
              title={isStreaming ? 'Pause' : 'Resume'}
            >
              {isStreaming ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={handleDownload} title="Download logs">
            <Download className="h-4 w-4" />
          </Button>
          {onClear && (
            <Button variant="ghost" size="sm" onClick={onClear} title="Clear logs">
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      <div className="relative flex-1 min-h-0">
        <ScrollArea ref={scrollRef} className="h-full" onScrollCapture={handleScroll}>
          <div className="p-4 font-mono text-sm bg-zinc-950 min-h-full">
            {logs.length === 0 ? (
              <div className="text-zinc-500 text-center py-8">No logs available</div>
            ) : (
              logs.map((line, index) => (
                <div
                  key={index}
                  className="py-0.5 text-zinc-300 hover:bg-zinc-900 whitespace-pre-wrap break-all"
                >
                  <span className="text-zinc-600 select-none mr-4 inline-block w-12 text-right">
                    {index + 1}
                  </span>
                  {formatLogLine(line)}
                </div>
              ))
            )}
          </div>
        </ScrollArea>

        {showScrollButton && (
          <Button
            variant="secondary"
            size="sm"
            className="absolute bottom-4 right-6 shadow-lg"
            onClick={scrollToBottom}
          >
            <ArrowDown className="h-4 w-4 mr-1" />
            Scroll to bottom
          </Button>
        )}
      </div>
    </div>
  )
}

function formatLogLine(line: string): React.ReactNode {
  if (line.match(/\berror\b/i)) {
    return <span className="text-red-400">{line}</span>
  }
  if (line.match(/\bwarn(ing)?\b/i)) {
    return <span className="text-yellow-400">{line}</span>
  }
  if (line.match(/\binfo\b/i)) {
    return <span className="text-blue-400">{line}</span>
  }
  if (line.match(/\bdebug\b/i)) {
    return <span className="text-zinc-500">{line}</span>
  }
  return line
}
