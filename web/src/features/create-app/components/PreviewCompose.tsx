import Editor from '@monaco-editor/react'
import { useTheme } from '@/shared/components/theme/ThemeProvider'

interface PreviewComposeProps {
    content: string
    height?: string
}

function PreviewCompose({ content, height = '300px' }: PreviewComposeProps) {
    const { actualTheme } = useTheme()
    
    return (
        <div className="border rounded-md overflow-hidden bg-muted">
            <Editor
                height={height}
                defaultLanguage="yaml"
                language="yaml"
                theme={actualTheme === 'dark' ? 'vs-dark' : 'light'}
                value={content}
                options={{
                    readOnly: true,
                    minimap: { enabled: false },
                    fontSize: 12,
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                }}
            />
        </div>
    )
}

export default PreviewCompose
