import Editor from '@monaco-editor/react'
import { useTheme } from '@/shared/components/theme/ThemeProvider'

function ComposeEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
    const { actualTheme } = useTheme()
    
    return (
        <div className="border rounded-md overflow-hidden">
            <Editor
                height="400px"
                defaultLanguage="yaml"
                language="yaml"
                theme={actualTheme === 'dark' ? 'vs-dark' : 'light'}
                value={value}
                onChange={(newValue) => onChange(newValue || '')}
                options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                }}
            />
        </div>
    )
}

export default ComposeEditor
