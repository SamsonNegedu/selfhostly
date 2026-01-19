import React, { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Save, RotateCcw, FileCode, AlertTriangle } from 'lucide-react'
import { useUpdateApp } from '@/shared/services/api'
import { useToast } from '@/shared/components/ui/Toast'

interface ComposeEditorProps {
    appId: string;
    initialComposeContent: string;
}

function ComposeEditor({ appId, initialComposeContent }: ComposeEditorProps) {
    const [composeContent, setComposeContent] = useState(initialComposeContent)
    const [isSaving, setIsSaving] = useState(false)
    const [hasChanges, setHasChanges] = useState(false)
    const [lineCount, setLineCount] = useState(0)
    const [charCount, setCharCount] = useState(0)
    const updateApp = useUpdateApp(appId)
    const { toast } = useToast()

    // Update line and character counts
    useEffect(() => {
        const lines = composeContent.split('\n').length
        setLineCount(lines)
        setCharCount(composeContent.length)
    }, [composeContent])

    const handleSave = async () => {
        if (!hasChanges) return

        setIsSaving(true)
        try {
            await updateApp.mutateAsync({
                compose_content: composeContent
            })
            setHasChanges(false)
            toast.success('Saved', 'Docker Compose configuration updated successfully')
        } catch (error) {
            console.error('Failed to update compose file:', error)
            toast.error('Failed to save', 'Could not update compose configuration')
        } finally {
            setIsSaving(false)
        }
    }

    const handleReset = () => {
        setComposeContent(initialComposeContent)
        setHasChanges(false)
        toast.info('Reset', 'Changes have been reset to the original version')
    }

    const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        const value = e.target.value
        setComposeContent(value)
        setHasChanges(value !== initialComposeContent)
    }

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between flex-wrap gap-4">
                    <div className="flex items-center gap-2">
                        <FileCode className="h-5 w-5 text-primary" />
                        <CardTitle className="text-xl">Docker Compose Editor</CardTitle>
                    </div>
                    <div className="flex gap-2 flex-wrap">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleReset}
                            disabled={!hasChanges}
                            title="Reset to original"
                            className="button-press"
                        >
                            <RotateCcw className="h-4 w-4 mr-2" />
                            Reset
                        </Button>
                        <Button
                            size="sm"
                            onClick={handleSave}
                            disabled={!hasChanges || isSaving}
                            className="button-press"
                        >
                            {isSaving ? (
                                <>
                                    <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                    Saving...
                                </>
                            ) : (
                                <>
                                    <Save className="h-4 w-4 mr-2" />
                                    Save Changes
                                </>
                            )}
                        </Button>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                <div className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                        Edit your docker-compose.yml file below. Use the Format button to auto-format your YAML.
                    </p>

                    {/* Editor container */}
                    <div className="relative">
                        {/* Line numbers */}
                        <div className="absolute left-0 top-0 bottom-0 w-12 bg-muted/30 border-r border-border overflow-hidden select-none">
                            <div className="font-mono text-xs text-muted-foreground text-right pr-2 pt-2">
                                {Array.from({ length: lineCount }, (_, i) => (
                                    <div key={i + 1} className="leading-6">
                                        {i + 1}
                                    </div>
                                ))}
                            </div>
                        </div>

                        {/* Textarea */}
                        <textarea
                            value={composeContent}
                            onChange={handleChange}
                            placeholder={`version: '3.8'
services:
  app:
    image: your-image:latest
    ports:
      - "80:80"
    restart: always`}
                            className="font-mono text-sm min-h-[500px] w-full pl-14 pr-4 py-2 rounded-md border focus:outline-none focus:ring-2 focus:ring-ring bg-background resize-y leading-6"
                            spellCheck={false}
                        />
                    </div>

                    {/* Status bar */}
                    <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t">
                        <div className="flex items-center gap-4">
                            <span>{lineCount} lines</span>
                            <span>{charCount} characters</span>
                        </div>
                        <div className="flex items-center gap-2">
                            {hasChanges && (
                                <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                                    <AlertTriangle className="h-3 w-3" />
                                    Unsaved changes
                                </span>
                            )}
                            <span>YAML</span>
                        </div>
                    </div>

                    {/* Tips */}
                    <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900/30 rounded-lg p-4">
                        <h4 className="font-medium text-sm mb-2 flex items-center gap-2">
                            <FileCode className="h-4 w-4" />
                            Tips for writing docker-compose.yml
                        </h4>
                        <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
                            <li>Use 2 spaces for indentation (not tabs)</li>
                            <li>Always specify the version at the top</li>
                            <li>Define services, volumes, and networks clearly</li>
                            <li>Use meaningful names for your services</li>
                            <li>Include restart policies for better reliability</li>
                        </ul>
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}

export default ComposeEditor
