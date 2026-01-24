import { useState } from 'react'

export interface Toast {
    id: string
    title: string
    description?: string
    variant?: 'default' | 'destructive' | 'success' | 'warning'
}

export const useToast = () => {
    const [toasts, setToasts] = useState<Toast[]>([])

    const addToast = (toast: Omit<Toast, 'id'>) => {
        const id = Math.random().toString(36).substr(2, 9)
        const newToast = { ...toast, id }
        setToasts((prev) => [...prev, newToast])

        // Auto-remove after 5 seconds
        setTimeout(() => {
            removeToast(id)
        }, 5000)
    }

    const removeToast = (id: string) => {
        setToasts((prev) => prev.filter((toast) => toast.id !== id))
    }

    const toast = {
        success: (title: string, description?: string) =>
            addToast({ title, description, variant: 'success' }),
        error: (title: string, description?: string) =>
            addToast({ title, description, variant: 'destructive' }),
        warning: (title: string, description?: string) =>
            addToast({ title, description, variant: 'warning' }),
        info: (title: string, description?: string) =>
            addToast({ title, description, variant: 'default' }),
    }

    return { toast, toasts, removeToast }
}

export function ToastContainer({ toasts, removeToast }: { toasts: Toast[]; removeToast: (id: string) => void }) {
    return (
        <div className="fixed bottom-4 right-4 z-50 space-y-2">
            {toasts.map((toast) => (
                <div
                    key={toast.id}
                    className={`max-w-sm p-4 rounded-lg shadow-lg border ${toast.variant === 'success'
                            ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-900 dark:text-green-100'
                            : toast.variant === 'destructive'
                                ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-900 dark:text-red-100'
                                : toast.variant === 'warning'
                                    ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800 text-yellow-900 dark:text-yellow-100'
                                    : 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800 text-blue-900 dark:text-blue-100'
                        } animate-in slide-in-from-right-2 duration-300 ease-out`}
                >
                    <div className="flex items-start">
                        <div className="flex-1">
                            <p className="font-medium">{toast.title}</p>
                            {toast.description && (
                                <p className="mt-1 text-sm opacity-80">{toast.description}</p>
                            )}
                        </div>
                        <button
                            onClick={() => removeToast(toast.id)}
                            className="ml-4 text-sm opacity-70 hover:opacity-100 focus:outline-none"
                        >
                            ✕
                        </button>
                    </div>
                </div>
            ))}
        </div>
    )
}
