export type HealthResponse = {
  server: string
  gadget_framework: string
  rbac: { status: string; stats?: unknown }
  timestamp: string
  version?: string
}


