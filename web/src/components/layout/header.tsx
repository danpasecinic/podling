import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { RefreshCw, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/hooks/use-auth'

interface HeaderProps {
  title: string
}

export function Header({ title }: HeaderProps) {
  const queryClient = useQueryClient()
  const { user, logout } = useAuth()

  const handleRefresh = () => queryClient.invalidateQueries()

  return (
    <header className="flex h-14 items-center gap-4 border-b bg-background px-4">
      <SidebarTrigger />
      <Separator orientation="vertical" className="h-6" />
      <h1 className="text-lg font-semibold">{title}</h1>
      <div className="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="icon" onClick={handleRefresh}>
          <RefreshCw className="h-4 w-4" />
        </Button>
        {user && (
          <>
            <Separator orientation="vertical" className="h-6" />
            <span className="text-sm text-muted-foreground">{user.username}</span>
            <Button variant="ghost" size="icon" onClick={logout}>
              <LogOut className="h-4 w-4" />
            </Button>
          </>
        )}
      </div>
    </header>
  )
}
