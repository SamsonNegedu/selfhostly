import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Textarea } from '@/shared/components/ui/textarea'
import { Save, RotateCcw } from 'lucide-react'
import { useUpdateApp } from '@/shared/services/api'

interface ComposeEditorProps {
    appId: string;
    initialComposeContent: string;
}

function ComposeEditor({ appId, initialComposeContent }: ComposeEditorProps) {
    const [composeContent, setComposeContent] = React.useState(initialComposeContent)
    const [isSaving, setIsSaving] = React.useState(false)
    const [hasChanges, setHasChanges] = React.useState(false)
    const updateApp = useUpdateApp(appId)

    const handleSave = async () => {
        if (!hasChanges) return

        setIsSaving(true)
        try {
            await updateApp.mutateAsync({
                compose_content: composeContent
            })
            setHasChanges(false)
        } catch (error) {
            console.error('Failed to update compose file:', error)
        } finally {
            setIsSaving(false)
        }
    }

    const handleReset = () => {
        setComposeContent(initialComposeContent)
        setHasChanges(false)
    }

    const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        setComposeContent(e.target.value)
        setHasChanges(e.target.value !== initialComposeContent)
    }

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <CardTitle className="text-xl">Docker Compose Editor</CardTitle>
                    <div className="flex gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleReset}
                            disabled={!hasChanges}
                            title="Reset to original"
                        >
                            <RotateCcw className="h-4 w-4 mr-2" />
                            Reset
                        </Button>
                        <Button
                            size="sm"
                            onClick={handleSave}
                            disabled={!hasChanges || isSaving}
                        >
                            <Save className="h-4 w-4 mr-2" />
                            {isSaving ? 'Saving...' : 'Save Changes'}
                        </Button>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                <div className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                        Edit the docker-compose.yml content below. The changes will be applied when you save.
                    </p>
                    <Textarea
                        value={composeContent}
                        onChange={handleChange}
                        placeholder="Enter docker-compose.yml content here..."
                        className="font-mono text-sm min-h-[400px]"
                    />
                    {hasChanges && (
                        <div className="text-sm text-foreground">
                            Unsaved changes
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}

export default ComposeEditor
