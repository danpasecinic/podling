export type NodeStatus = 'online' | 'offline'
export type TaskStatus = 'pending' | 'scheduled' | 'running' | 'completed' | 'failed'
export type PodStatus = 'pending' | 'scheduled' | 'running' | 'succeeded' | 'failed' | 'unknown'
export type ContainerStatus = 'waiting' | 'running' | 'terminated'
export type HealthStatus = 'healthy' | 'unhealthy' | 'unknown'
export type ProbeType = 'http' | 'tcp' | 'exec'
export type RestartPolicy = 'Always' | 'OnFailure' | 'Never'
export type ServiceType = 'ClusterIP' | 'NodePort' | 'LoadBalancer'

export interface ResourceList {
  cpu: number
  memory: number
}

export interface NodeResources {
  capacity: ResourceList
  allocatable: ResourceList
  used: ResourceList
}

export interface Node {
  nodeId: string
  hostname: string
  port: number
  status: NodeStatus
  runningTasks: number
  lastHeartbeat: string
  resources: NodeResources | null
}

export interface HealthCheck {
  type: ProbeType
  httpPath?: string
  port?: number
  command?: string[]
  initialDelaySeconds?: number
  periodSeconds?: number
  timeoutSeconds?: number
  successThreshold?: number
  failureThreshold?: number
}

export interface ContainerPort {
  name?: string
  containerPort: number
  protocol?: string
  hostPort?: number
}

export interface ResourceRequirements {
  requests?: ResourceList
  limits?: ResourceList
}

export interface Task {
  taskId: string
  name: string
  image: string
  env?: Record<string, string>
  status: TaskStatus
  nodeId?: string
  containerId?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
  error?: string
  livenessProbe?: HealthCheck
  readinessProbe?: HealthCheck
  restartPolicy?: RestartPolicy
  healthStatus: HealthStatus
  resources?: ResourceRequirements
  ports?: ContainerPort[]
}

export interface Container {
  name: string
  image: string
  command?: string[]
  args?: string[]
  env?: Record<string, string>
  ports?: ContainerPort[]
  livenessProbe?: HealthCheck
  readinessProbe?: HealthCheck
  workingDir?: string
  resources?: ResourceRequirements
  containerId?: string
  status: ContainerStatus
  healthStatus: HealthStatus
  startedAt?: string
  finishedAt?: string
  exitCode?: number
  error?: string
  restartCount?: number
}

export interface Pod {
  podId: string
  name: string
  namespace: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  containers: Container[]
  status: PodStatus
  nodeId?: string
  restartPolicy?: RestartPolicy
  createdAt: string
  scheduledAt?: string
  startedAt?: string
  finishedAt?: string
  message?: string
  reason?: string
}

export interface ServicePort {
  name?: string
  protocol?: string
  port: number
  targetPort?: number
  nodePort?: number
}

export interface EndpointAddress {
  ip: string
  podId: string
  nodeId?: string
}

export interface EndpointSubset {
  addresses: EndpointAddress[]
  notReadyAddresses?: EndpointAddress[]
  ports: ServicePort[]
}

export interface Endpoints {
  serviceId: string
  serviceName: string
  namespace: string
  subsets: EndpointSubset[]
  updatedAt: string
}

export interface Service {
  serviceId: string
  name: string
  namespace: string
  type: ServiceType
  clusterIp: string
  selector?: Record<string, string>
  ports: ServicePort[]
  labels?: Record<string, string>
  annotations?: Record<string, string>
  sessionAffinity?: string
  createdAt: string
  updatedAt: string
}

export interface PruneResult {
  podsRemoved: number
  nodesRemoved: number
  servicesRemoved: number
  tasksRemoved: number
}
