import { Header } from '@/components/layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function Settings() {
  return (
    <>
      <Header title="Settings" />
      <main className="flex-1 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
            <CardDescription>Configure dashboard preferences</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Settings page coming soon. For now, the dashboard auto-refreshes every 5 seconds.
            </p>
          </CardContent>
        </Card>
      </main>
    </>
  )
}
