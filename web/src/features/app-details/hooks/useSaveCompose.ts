import { useState } from 'react'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { useUpdateApp, useUpdateAppContainers } from '@/shared/services/api'

// What the toasts say, since the same save is used for the whole compose file and for just its variables.
interface SaveCopy {
    redeploying: string
    saved: string
}

// Saves an app's compose file, and optionally restarts its containers so the change takes effect now. Both editors
// of the compose file save through this, so they behave the same and cannot drift apart. `onSaved` runs as soon as the
// file is stored, before any restart.
export function useSaveCompose(appId: string, nodeId: string, copy: SaveCopy) {
    const updateApp = useUpdateApp(appId, nodeId)
    const updateContainers = useUpdateAppContainers()
    const { toast } = useToast()
    const [saving, setSaving] = useState(false)

    const save = async (content: string, redeploy: boolean, onSaved?: () => void) => {
        setSaving(true)
        try {
            await updateApp.mutateAsync({ compose_content: content })
            onSaved?.()
            if (redeploy) {
                await updateContainers.mutateAsync({ id: appId, nodeId })
                toast.success('Saved and redeploying', copy.redeploying)
            } else {
                toast.success('Saved', copy.saved)
            }
        } catch (error) {
            toast.error('Could not save', describeError(error))
        } finally {
            setSaving(false)
        }
    }

    return { save, saving }
}
