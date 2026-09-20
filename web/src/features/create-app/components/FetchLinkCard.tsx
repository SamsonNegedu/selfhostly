import { Loader2 } from 'lucide-react'
import { Button } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { EmptyState } from '@/shared/components/ui/EmptyState'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { YamlEditor } from '@/shared/components/ui/YamlEditor'
import type { FetchState } from '../hooks/useNewAppForm'

interface FetchLinkCardProps {
    link: string
    onLinkChange: (link: string) => void
    fetched: FetchState
    onFetch: () => Promise<void>
    // The file that came back, which can still be edited before deploying.
    fetchedText: string
    onFetchedTextChange: (text: string) => void
}

// Loads a compose file from a link, such as a GitHub file page, and shows it for editing.
function FetchLinkCard({ link, onLinkChange, fetched, onFetch, fetchedText, onFetchedTextChange }: FetchLinkCardProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-base">Fetch a compose file</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
                <form
                    className="flex flex-col gap-3 sm:flex-row sm:items-end"
                    onSubmit={(event) => {
                        event.preventDefault()
                        if (link.trim() !== '') void onFetch()
                    }}
                >
                    <Field
                        label="Link to the file"
                        hint="A GitHub file page or a raw address, for example github.com/you/repo/blob/main/docker-compose.yml"
                        className="flex-1"
                    >
                        <Input
                            value={link}
                            onChange={(event) => onLinkChange(event.target.value)}
                            placeholder="https://github.com/..."
                            inputMode="url"
                        />
                    </Field>
                    <Button type="submit" disabled={link.trim() === '' || fetched.status === 'loading'}>
                        {fetched.status === 'loading' && (
                            <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                        )}
                        Fetch
                    </Button>
                </form>
                {fetched.status === 'error' && (
                    <p role="alert" className="text-sm text-status-err-fg">
                        {fetched.message}
                    </p>
                )}
                {fetched.status === 'ok' && (
                    <>
                        <p role="status" className="text-sm text-status-ok-fg">
                            Fetched. You can still edit it before deploying.
                        </p>
                        <YamlEditor
                            value={fetchedText}
                            onChange={onFetchedTextChange}
                            aria-label="Fetched compose file"
                            height={320}
                        />
                    </>
                )}
                {fetched.status === 'idle' && (
                    <EmptyState
                        title="Nothing fetched yet"
                        description="Private repositories and links that need a login cannot be fetched from the browser. Paste the file instead."
                        className="py-8"
                    />
                )}
            </CardContent>
        </Card>
    )
}

export default FetchLinkCard
