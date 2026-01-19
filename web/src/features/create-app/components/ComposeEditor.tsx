import React from 'react'
import Editor from '@monaco-editor/react'

function ComposeEditor({ value, onChange }: { value: string; onChange: (value: string) => void }) {
    return (
        <div className="border rounded-md overflow-hidden">
            <Editor
                height="400px"
                defaultLanguage="yaml"
                language="yaml"
                theme="vs-dark"
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
