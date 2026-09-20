export type StatusKind = 'ok' | 'warn' | 'err' | 'info' | 'idle'

export interface StatusMeta {
    kind: StatusKind
    label: string
}

const FALLBACK_STATUS: StatusMeta = { kind: 'idle', label: 'Unknown' }

const APP_STATUS: Record<string, StatusMeta> = {
    running: { kind: 'ok', label: 'Running' },
    stopped: { kind: 'idle', label: 'Stopped' },
    updating: { kind: 'info', label: 'Updating' },
    pending: { kind: 'info', label: 'Pending' },
    error: { kind: 'err', label: 'Failed' },
}

const NODE_STATUS: Record<string, StatusMeta> = {
    online: { kind: 'ok', label: 'Online' },
    offline: { kind: 'warn', label: 'Offline' },
    unreachable: { kind: 'warn', label: 'Unreachable' },
    unknown: { kind: 'idle', label: 'Checking' },
    error: { kind: 'err', label: 'Error' },
}

const JOB_STATUS: Record<string, StatusMeta> = {
    pending: { kind: 'info', label: 'Pending' },
    running: { kind: 'info', label: 'Running' },
    completed: { kind: 'ok', label: 'Completed' },
    failed: { kind: 'err', label: 'Failed' },
}

export const appStatusMeta = (status: string): StatusMeta => APP_STATUS[status] ?? FALLBACK_STATUS
export const nodeStatusMeta = (status: string): StatusMeta => NODE_STATUS[status] ?? FALLBACK_STATUS
export const jobStatusMeta = (status: string): StatusMeta => JOB_STATUS[status] ?? FALLBACK_STATUS
