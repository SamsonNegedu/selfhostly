import Editor from '@monaco-editor/react'

function PreviewCompose({ content }: { content: string }) {
    return (
        <div className="border rounded-md overflow-hidden bg-muted">
            <Editor
                height="300px"
                defaultLanguage="yaml"
                language="yaml"
                theme="vs-dark"
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
