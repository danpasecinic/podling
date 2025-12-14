import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { useCreateService } from '@/hooks'

interface PortSpec {
  name: string
  port: string
  targetPort: string
}

interface SelectorPair {
  key: string
  value: string
}

export function CreateServiceDialog() {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [namespace, setNamespace] = useState('default')
  const [selectors, setSelectors] = useState<SelectorPair[]>([{ key: '', value: '' }])
  const [ports, setPorts] = useState<PortSpec[]>([{ name: '', port: '80', targetPort: '' }])

  const createService = useCreateService()

  const handleAddSelector = () => {
    setSelectors([...selectors, { key: '', value: '' }])
  }

  const handleRemoveSelector = (index: number) => {
    if (selectors.length > 1) {
      setSelectors(selectors.filter((_, i) => i !== index))
    }
  }

  const handleSelectorChange = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...selectors]
    updated[index][field] = value
    setSelectors(updated)
  }

  const handleAddPort = () => {
    setPorts([...ports, { name: '', port: '', targetPort: '' }])
  }

  const handleRemovePort = (index: number) => {
    if (ports.length > 1) {
      setPorts(ports.filter((_, i) => i !== index))
    }
  }

  const handlePortChange = (index: number, field: keyof PortSpec, value: string) => {
    const updated = [...ports]
    updated[index][field] = value
    setPorts(updated)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const selectorMap: Record<string, string> = {}
    selectors.forEach((s) => {
      if (s.key.trim()) {
        selectorMap[s.key.trim()] = s.value
      }
    })

    const portSpecs = ports
      .filter((p) => p.port)
      .map((p) => ({
        name: p.name.trim() || undefined,
        port: parseInt(p.port, 10),
        targetPort: p.targetPort ? parseInt(p.targetPort, 10) : parseInt(p.port, 10),
      }))

    createService.mutate(
      {
        name: name.trim(),
        namespace: namespace.trim() || 'default',
        selector: Object.keys(selectorMap).length > 0 ? selectorMap : undefined,
        ports: portSpecs,
      },
      {
        onSuccess: () => {
          setOpen(false)
          resetForm()
        },
      }
    )
  }

  const resetForm = () => {
    setName('')
    setNamespace('default')
    setSelectors([{ key: '', value: '' }])
    setPorts([{ name: '', port: '80', targetPort: '' }])
  }

  const isValid =
    name.trim() &&
    ports.some((p) => p.port && !isNaN(parseInt(p.port, 10)))

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Service
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[550px] max-h-[80vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Service</DialogTitle>
            <DialogDescription>
              Create a service to expose pods with a stable endpoint.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="service-name">Name</Label>
                <Input
                  id="service-name"
                  placeholder="my-service"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="service-namespace">Namespace</Label>
                <Input
                  id="service-namespace"
                  placeholder="default"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                />
              </div>
            </div>

            <Separator />

            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>Pod Selector</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddSelector}>
                  <Plus className="mr-1 h-3 w-3" />
                  Add
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Select pods by their labels (e.g., app=nginx)
              </p>
              {selectors.map((selector, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    placeholder="key (e.g., app)"
                    value={selector.key}
                    onChange={(e) => handleSelectorChange(index, 'key', e.target.value)}
                  />
                  <Input
                    placeholder="value (e.g., nginx)"
                    value={selector.value}
                    onChange={(e) => handleSelectorChange(index, 'value', e.target.value)}
                  />
                  {selectors.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveSelector(index)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
            </div>

            <Separator />

            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>Ports</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddPort}>
                  <Plus className="mr-1 h-3 w-3" />
                  Add
                </Button>
              </div>
              {ports.map((port, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    placeholder="Name (optional)"
                    value={port.name}
                    onChange={(e) => handlePortChange(index, 'name', e.target.value)}
                    className="w-28"
                  />
                  <Input
                    placeholder="Port"
                    type="number"
                    value={port.port}
                    onChange={(e) => handlePortChange(index, 'port', e.target.value)}
                    className="w-20"
                  />
                  <Input
                    placeholder="Target"
                    type="number"
                    value={port.targetPort}
                    onChange={(e) => handlePortChange(index, 'targetPort', e.target.value)}
                    className="w-20"
                  />
                  {ports.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemovePort(index)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
              <p className="text-xs text-muted-foreground">
                Port: service port, Target: container port (defaults to service port)
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!isValid || createService.isPending}>
              {createService.isPending ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
