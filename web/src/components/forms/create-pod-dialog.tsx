import { useState } from 'react'
import { Container, Plus, X } from 'lucide-react'
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
import { useCreatePod } from '@/hooks'

interface ContainerSpec {
  name: string
  image: string
  env: { key: string; value: string }[]
}

interface LabelPair {
  key: string
  value: string
}

export function CreatePodDialog() {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [namespace, setNamespace] = useState('default')
  const [labels, setLabels] = useState<LabelPair[]>([])
  const [containers, setContainers] = useState<ContainerSpec[]>([
    { name: '', image: '', env: [] },
  ])

  const createPod = useCreatePod()

  const handleAddContainer = () => {
    setContainers([...containers, { name: '', image: '', env: [] }])
  }

  const handleRemoveContainer = (index: number) => {
    if (containers.length > 1) {
      setContainers(containers.filter((_, i) => i !== index))
    }
  }

  const handleContainerChange = (index: number, field: 'name' | 'image', value: string) => {
    const updated = [...containers]
    updated[index][field] = value
    setContainers(updated)
  }

  const handleAddContainerEnv = (containerIndex: number) => {
    const updated = [...containers]
    updated[containerIndex].env.push({ key: '', value: '' })
    setContainers(updated)
  }

  const handleRemoveContainerEnv = (containerIndex: number, envIndex: number) => {
    const updated = [...containers]
    updated[containerIndex].env = updated[containerIndex].env.filter((_, i) => i !== envIndex)
    setContainers(updated)
  }

  const handleContainerEnvChange = (
    containerIndex: number,
    envIndex: number,
    field: 'key' | 'value',
    value: string
  ) => {
    const updated = [...containers]
    updated[containerIndex].env[envIndex][field] = value
    setContainers(updated)
  }

  const handleAddLabel = () => {
    setLabels([...labels, { key: '', value: '' }])
  }

  const handleRemoveLabel = (index: number) => {
    setLabels(labels.filter((_, i) => i !== index))
  }

  const handleLabelChange = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...labels]
    updated[index][field] = value
    setLabels(updated)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const labelMap: Record<string, string> = {}
    labels.forEach((l) => {
      if (l.key.trim()) {
        labelMap[l.key.trim()] = l.value
      }
    })

    const containerSpecs = containers.map((c) => {
      const env: Record<string, string> = {}
      c.env.forEach((e) => {
        if (e.key.trim()) {
          env[e.key.trim()] = e.value
        }
      })
      return {
        name: c.name.trim(),
        image: c.image.trim(),
        env: Object.keys(env).length > 0 ? env : undefined,
      }
    })

    createPod.mutate(
      {
        name: name.trim(),
        namespace: namespace.trim() || 'default',
        labels: Object.keys(labelMap).length > 0 ? labelMap : undefined,
        containers: containerSpecs,
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
    setLabels([])
    setContainers([{ name: '', image: '', env: [] }])
  }

  const isValid =
    name.trim() &&
    containers.every((c) => c.name.trim() && c.image.trim())

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Pod
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Pod</DialogTitle>
            <DialogDescription>
              Create a new multi-container pod.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="pod-name">Name</Label>
                <Input
                  id="pod-name"
                  placeholder="my-pod"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="namespace">Namespace</Label>
                <Input
                  id="namespace"
                  placeholder="default"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                />
              </div>
            </div>

            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>Labels</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddLabel}>
                  <Plus className="mr-1 h-3 w-3" />
                  Add
                </Button>
              </div>
              {labels.map((label, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    placeholder="key"
                    value={label.key}
                    onChange={(e) => handleLabelChange(index, 'key', e.target.value)}
                  />
                  <Input
                    placeholder="value"
                    value={label.value}
                    onChange={(e) => handleLabelChange(index, 'value', e.target.value)}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => handleRemoveLabel(index)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>

            <Separator />

            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <Label className="text-base font-semibold">Containers</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddContainer}>
                  <Container className="mr-1 h-3 w-3" />
                  Add Container
                </Button>
              </div>

              {containers.map((container, cIndex) => (
                <div key={cIndex} className="rounded-lg border p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">Container {cIndex + 1}</span>
                    {containers.length > 1 && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRemoveContainer(cIndex)}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="grid gap-1">
                      <Label className="text-xs">Name</Label>
                      <Input
                        placeholder="nginx"
                        value={container.name}
                        onChange={(e) => handleContainerChange(cIndex, 'name', e.target.value)}
                      />
                    </div>
                    <div className="grid gap-1">
                      <Label className="text-xs">Image</Label>
                      <Input
                        placeholder="nginx:latest"
                        value={container.image}
                        onChange={(e) => handleContainerChange(cIndex, 'image', e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <div className="flex items-center justify-between">
                      <Label className="text-xs">Environment Variables</Label>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleAddContainerEnv(cIndex)}
                      >
                        <Plus className="h-3 w-3" />
                      </Button>
                    </div>
                    {container.env.map((env, eIndex) => (
                      <div key={eIndex} className="flex gap-2">
                        <Input
                          placeholder="KEY"
                          value={env.key}
                          onChange={(e) =>
                            handleContainerEnvChange(cIndex, eIndex, 'key', e.target.value)
                          }
                          className="font-mono text-xs"
                        />
                        <Input
                          placeholder="value"
                          value={env.value}
                          onChange={(e) =>
                            handleContainerEnvChange(cIndex, eIndex, 'value', e.target.value)
                          }
                          className="text-xs"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          onClick={() => handleRemoveContainerEnv(cIndex, eIndex)}
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!isValid || createPod.isPending}>
              {createPod.isPending ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
