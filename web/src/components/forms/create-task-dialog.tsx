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
import { useCreateTask } from '@/hooks'

interface EnvVar {
  key: string
  value: string
}

export function CreateTaskDialog() {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [image, setImage] = useState('')
  const [envVars, setEnvVars] = useState<EnvVar[]>([])

  const createTask = useCreateTask()

  const handleAddEnvVar = () => {
    setEnvVars([...envVars, { key: '', value: '' }])
  }

  const handleRemoveEnvVar = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index))
  }

  const handleEnvVarChange = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...envVars]
    updated[index][field] = value
    setEnvVars(updated)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const env: Record<string, string> = {}
    envVars.forEach((v) => {
      if (v.key.trim()) {
        env[v.key.trim()] = v.value
      }
    })

    createTask.mutate(
      {
        name: name.trim(),
        image: image.trim(),
        env: Object.keys(env).length > 0 ? env : undefined,
      },
      {
        onSuccess: () => {
          setOpen(false)
          setName('')
          setImage('')
          setEnvVars([])
        },
      }
    )
  }

  const isValid = name.trim() && image.trim()

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Task
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Task</DialogTitle>
            <DialogDescription>
              Create a new single-container task.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="my-nginx"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="image">Image</Label>
              <Input
                id="image"
                placeholder="nginx:latest"
                value={image}
                onChange={(e) => setImage(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>Environment Variables</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddEnvVar}>
                  <Plus className="mr-1 h-3 w-3" />
                  Add
                </Button>
              </div>
              {envVars.map((env, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    placeholder="KEY"
                    value={env.key}
                    onChange={(e) => handleEnvVarChange(index, 'key', e.target.value)}
                    className="font-mono"
                  />
                  <Input
                    placeholder="value"
                    value={env.value}
                    onChange={(e) => handleEnvVarChange(index, 'value', e.target.value)}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => handleRemoveEnvVar(index)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!isValid || createTask.isPending}>
              {createTask.isPending ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
