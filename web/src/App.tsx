import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
import { AuthProvider } from '@/hooks'
import { ProtectedRoute } from '@/components/auth'
import { AppShell } from '@/components/layout'
import {
  Login,
  NodeDetail,
  Nodes,
  Overview,
  PodDetail,
  PodLogs,
  Pods,
  ServiceDetail,
  Services,
  Settings,
  Signup,
  TaskDetail,
  TaskLogs,
  Tasks,
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
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/signup" element={<Signup />} />
              <Route
                element={
                  <ProtectedRoute>
                    <AppShell />
                  </ProtectedRoute>
                }
              >
                <Route path="/" element={<Overview />} />
                <Route path="/nodes" element={<Nodes />} />
                <Route path="/nodes/:nodeId" element={<NodeDetail />} />
                <Route path="/pods" element={<Pods />} />
                <Route path="/pods/:podId" element={<PodDetail />} />
                <Route path="/pods/:podId/logs" element={<PodLogs />} />
                <Route path="/tasks" element={<Tasks />} />
                <Route path="/tasks/:taskId" element={<TaskDetail />} />
                <Route path="/tasks/:taskId/logs" element={<TaskLogs />} />
                <Route path="/services" element={<Services />} />
                <Route path="/services/:serviceId" element={<ServiceDetail />} />
                <Route path="/settings" element={<Settings />} />
              </Route>
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </TooltipProvider>
      <Toaster />
    </QueryClientProvider>
  )
}

export default App
