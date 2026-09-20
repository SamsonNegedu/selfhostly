import { create } from 'zustand'
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X, type LucideIcon } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import type { StatusKind } from '@/shared/lib/status'
import { Button } from './Button'

const TOAST_LIFETIME_MS = 5000
const ID_RADIX = 36
const ID_START = 2
const ID_END = 11

interface ToastAction {
    label: string
    onClick: () => void
}

export interface Toast {
    id: string
    title: string
    description?: string
    variant?: 'default' | 'destructive' | 'success' | 'warning'
    action?: ToastAction
}

interface ToastStore {
    toasts: Toast[]
    addToast: (toast: Omit<Toast, 'id'>) => void
    removeToast: (id: string) => void
}

// One list for the whole app. Feature code fires toasts from anywhere and the single container in the
// app shell shows them. Keeping the list per component would hide every toast that is not rendered locally.
const useToastStore = create<ToastStore>((set, get) => ({
    toasts: [],
    addToast: (toast) => {
        const id = Math.random().toString(ID_RADIX).substring(ID_START, ID_END)
        set((state) => ({ toasts: [...state.toasts, { ...toast, id }] }))
        setTimeout(() => get().removeToast(id), TOAST_LIFETIME_MS)
    },
    removeToast: (id) => set((state) => ({ toasts: state.toasts.filter((toast) => toast.id !== id) })),
}))

const toast = {
    success: (title: string, description?: string, action?: ToastAction) =>
        useToastStore.getState().addToast({ title, description, action, variant: 'success' }),
    error: (title: string, description?: string, action?: ToastAction) =>
        useToastStore.getState().addToast({ title, description, action, variant: 'destructive' }),
    warning: (title: string, description?: string, action?: ToastAction) =>
        useToastStore.getState().addToast({ title, description, action, variant: 'warning' }),
    info: (title: string, description?: string, action?: ToastAction) =>
        useToastStore.getState().addToast({ title, description, action, variant: 'default' }),
}

export const useToast = () => {
    const toasts = useToastStore((state) => state.toasts)
    const removeToast = useToastStore((state) => state.removeToast)
    return { toast, toasts, removeToast }
}

const VARIANT_STYLE: Record<NonNullable<Toast['variant']>, { kind: StatusKind; Icon: LucideIcon }> = {
    default: { kind: 'info', Icon: Info },
    success: { kind: 'ok', Icon: CheckCircle2 },
    warning: { kind: 'warn', Icon: AlertTriangle },
    destructive: { kind: 'err', Icon: AlertCircle },
}

const ICON_TILE_CLASSES: Record<StatusKind, string> = {
    ok: 'bg-status-ok-bg text-status-ok-fg',
    warn: 'bg-status-warn-bg text-status-warn-fg',
    err: 'bg-status-err-bg text-status-err-fg',
    info: 'bg-status-info-bg text-status-info-fg',
    idle: 'bg-status-idle-bg text-status-idle-fg',
}

export function ToastContainer({ toasts, removeToast }: { toasts: Toast[]; removeToast: (id: string) => void }) {
    return (
        <div
            className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-md:bottom-auto max-md:left-4 max-md:top-[68px]"
            aria-live="polite"
        >
            {toasts.map((toast) => {
                const { kind, Icon } = VARIANT_STYLE[toast.variant ?? 'default']
                return (
                    <div
                        key={toast.id}
                        role={toast.variant === 'destructive' ? 'alert' : 'status'}
                        className="flex w-[360px] max-w-full items-start gap-3 rounded-xl border border-border bg-card p-3.5 text-card-foreground shadow-xl"
                    >
                        <div
                            aria-hidden="true"
                            className={cn(
                                'flex h-7 w-7 shrink-0 items-center justify-center rounded-lg',
                                ICON_TILE_CLASSES[kind],
                            )}
                        >
                            <Icon className="h-4 w-4" />
                        </div>
                        <div className="flex min-w-0 flex-1 flex-col">
                            <p className="text-[13.5px] font-semibold">{toast.title}</p>
                            {toast.description && (
                                <p className="text-[12.5px] text-muted-foreground">{toast.description}</p>
                            )}
                        </div>
                        {toast.action && (
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    toast.action?.onClick()
                                    removeToast(toast.id)
                                }}
                            >
                                {toast.action.label}
                            </Button>
                        )}
                        <Button
                            variant="ghost"
                            size="icon"
                            className="-my-1 -mr-1.5 h-7 w-7"
                            onClick={() => removeToast(toast.id)}
                            aria-label="Dismiss notification"
                        >
                            <X className="h-4 w-4" />
                        </Button>
                    </div>
                )
            })}
        </div>
    )
}
