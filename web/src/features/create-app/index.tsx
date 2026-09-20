import { Link } from 'react-router-dom'
import { ClipboardPaste, LayoutGrid, Link2, Loader2, Rocket } from 'lucide-react'
import ActionBar from '@/shared/components/ui/ActionBar'
import { Button, buttonClasses } from '@/shared/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { SegmentedControl } from '@/shared/components/ui/SegmentedControl'
import { YamlEditor } from '@/shared/components/ui/YamlEditor'
import { describeError } from '@/shared/lib/errors'
import { ROUTES } from '@/shared/lib/routes'
import ComposeChecks from '@/features/app-details/components/ComposeChecks'
import ConfigureCard from './components/ConfigureCard'
import FetchLinkCard from './components/FetchLinkCard'
import PasteComposeCard from './components/PasteComposeCard'
import TemplateGrid from './components/TemplateGrid'
import { useNewAppForm } from './hooks/useNewAppForm'
import type { Mode } from './lib/new-app-rules'

const MODE_OPTIONS = [
    { value: 'template', label: 'Templates', icon: <LayoutGrid className="h-4 w-4" /> },
    { value: 'paste', label: 'Paste compose', icon: <ClipboardPaste className="h-4 w-4" /> },
    { value: 'link', label: 'From a link', icon: <Link2 className="h-4 w-4" /> },
]

function NewApp() {
    const form = useNewAppForm()
    const { mode, template, checks, blocked } = form

    return (
        <div className="flex flex-col gap-5">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">New app</h1>
                    <p className="text-muted-foreground">Pick a template, or bring your own compose file.</p>
                </div>
                <Link to={ROUTES.fleet} className={buttonClasses({ variant: 'ghost' })}>
                    Cancel
                </Link>
            </div>

            <SegmentedControl
                aria-label="How to start"
                options={MODE_OPTIONS}
                value={mode}
                onValueChange={(value) => form.setMode(value as Mode)}
                className="self-start"
            />

            <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
                <div className="flex min-w-0 flex-col gap-5">
                    {mode === 'template' && (
                        <TemplateGrid templateId={form.templateId} onSelect={form.selectTemplate} />
                    )}

                    {mode === 'paste' && <PasteComposeCard value={form.pasted} onChange={form.setPasted} />}

                    {mode === 'link' && (
                        <FetchLinkCard
                            link={form.link}
                            onLinkChange={form.setLink}
                            fetched={form.fetched}
                            onFetch={form.fetchLink}
                            fetchedText={form.pasted}
                            onFetchedTextChange={form.setPasted}
                        />
                    )}

                    {form.showConfigure && <ConfigureCard form={form} />}
                </div>

                <div className="flex min-w-0 flex-col gap-5 lg:sticky lg:top-4 lg:self-start">
                    {mode === 'template' && template && (
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">Compose file</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <YamlEditor
                                    value={form.content}
                                    readOnly
                                    aria-label="Generated compose file"
                                    height={260}
                                />
                                <p className="mt-2 text-compact text-muted-foreground">
                                    Generated from the template. You can edit it after the app is created.
                                </p>
                            </CardContent>
                        </Card>
                    )}
                    {checks.length > 0 && <ComposeChecks checks={checks} />}
                </div>
            </div>

            {form.createError && (
                <p role="alert" className="rounded-lg bg-status-err-bg px-3.5 py-3 text-sm text-status-err-fg">
                    Could not create the app. {describeError(form.createError)}
                </p>
            )}

            <ActionBar label="Create">
                <span className="text-sm font-medium">
                    {blocked ?? `Ready to deploy ${form.name} to ${form.nodeName}`}
                </span>
                <div className="flex items-center gap-2">
                    <Link to={ROUTES.fleet} className={buttonClasses({ variant: 'ghost' })}>
                        Cancel
                    </Link>
                    <Button onClick={form.create} disabled={!!blocked || form.creating}>
                        {form.creating ? (
                            <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
                        ) : (
                            <Rocket className="h-4 w-4" />
                        )}
                        Create app
                    </Button>
                </div>
            </ActionBar>
        </div>
    )
}

export default NewApp
