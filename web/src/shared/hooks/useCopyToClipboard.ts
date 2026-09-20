import { useToast } from '@/shared/components/ui/Toast'

// Puts text on the clipboard and says so. `successDescription` is what the toast says on success. If the browser
// refuses, the toast says that instead.
export function useCopyToClipboard() {
    const { toast } = useToast()

    return async (text: string, successDescription: string) => {
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied', successDescription)
        } catch {
            toast.error('Could not copy', 'Your browser blocked access to the clipboard')
        }
    }
}
