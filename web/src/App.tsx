import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppShell } from '@/components/layout'
import {
  Overview,
  Nodes,
  NodeDetail,
  Pods,
  PodDetail,
  Tasks,
  TaskDetail,
  Services,
  ServiceDetail,
  Settings,
} from '@/pages'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 3000,
      retry: 1,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<AppShell />}>
              <Route path="/" element={<Overview />} />
              <Route path="/nodes" element={<Nodes />} />
              <Route path="/nodes/:nodeId" element={<NodeDetail />} />
              <Route path="/pods" element={<Pods />} />
              <Route path="/pods/:podId" element={<PodDetail />} />
              <Route path="/tasks" element={<Tasks />} />
              <Route path="/tasks/:taskId" element={<TaskDetail />} />
              <Route path="/services" element={<Services />} />
              <Route path="/services/:serviceId" element={<ServiceDetail />} />
              <Route path="/settings" element={<Settings />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

export default App
