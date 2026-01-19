import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCreateApp } from '@/shared/services/api'
import { Button } from '@/shared/components/ui'
import ComposeEditor from './components/ComposeEditor'
import PreviewCompose from './components/PreviewCompose'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'

function CreateApp() {
    const navigate = useNavigate()
    const createApp = useCreateApp()
    const [step, setStep] = useState(1)
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        compose_content: '',
    })

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        createApp.mutate(formData, {
            onSuccess: () => {
                navigate('/dashboard')
            },
        })
    }

    return (
        <div>
            {/* Breadcrumb Navigation */}
            <div className="mb-6">
                <AppBreadcrumb
                    items={[
                        { label: 'Home', path: '/dashboard' },
                        { label: 'Apps', path: '/apps' },
                        { label: 'New App', isCurrentPage: true }
                    ]}
                />
            </div>

            <div className="mb-6">
                <h1 className="text-3xl font-bold">Create New App</h1>
                <p className="text-muted-foreground mt-2">
                    Deploy a new self-hosted application
                </p>
            </div>

            <div className="max-w-3xl">
                {step === 1 && (
                    <div className="space-y-4">
                        <h2 className="text-2xl font-semibold">App Information</h2>
                        <div>
                            <label htmlFor="name" className="block text-sm font-medium mb-2">
                                App Name
                            </label>
                            <input
                                id="name"
                                type="text"
                                value={formData.name}
                                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                                placeholder="my-app"
                                required
                            />
                        </div>
                        <div>
                            <label htmlFor="description" className="block text-sm font-medium mb-2">
                                Description (optional)
                            </label>
                            <textarea
                                id="description"
                                value={formData.description}
                                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-ring min-h-[100px]"
                                placeholder="A brief description of your app"
                            />
                        </div>
                        <Button onClick={() => setStep(2)}>Next</Button>
                    </div>
                )}

                {step === 2 && (
                    <div className="space-y-4">
                        <h2 className="text-2xl font-semibold">Docker Compose</h2>
                        <ComposeEditor
                            value={formData.compose_content}
                            onChange={(value) => setFormData({ ...formData, compose_content: value })}
                        />
                        <div className="flex gap-2">
                            <Button variant="outline" onClick={() => setStep(1)}>
                                Back
                            </Button>
                            <Button onClick={() => setStep(3)}>Next</Button>
                        </div>
                    </div>
                )}

                {step === 3 && (
                    <div className="space-y-4">
                        <h2 className="text-2xl font-semibold">Review & Deploy</h2>
                        <div className="space-y-4">
                            <div>
                                <h3 className="font-semibold mb-2">App Name</h3>
                                <p>{formData.name}</p>
                            </div>
                            <div>
                                <h3 className="font-semibold mb-2">Description</h3>
                                <p>{formData.description || 'No description'}</p>
                            </div>
                            <div>
                                <h3 className="font-semibold mb-2">Cloudflare Tunnel</h3>
                                <p>Will be automatically configured</p>
                            </div>
                            <div>
                                <h3 className="font-semibold mb-2">Docker Compose</h3>
                                <PreviewCompose content={formData.compose_content} />
                            </div>
                        </div>
                        <div className="flex gap-2">
                            <Button variant="outline" onClick={() => setStep(2)}>
                                Back
                            </Button>
                            <Button onClick={handleSubmit} disabled={createApp.isPending}>
                                {createApp.isPending ? 'Creating...' : 'Deploy App'}
                            </Button>
                        </div>
                        {createApp.error && (
                            <p className="text-destructive text-sm mt-2">
                                {createApp.error.message}
                            </p>
                        )}
                    </div>
                )}
            </div>
        </div>
    )
}

export default CreateApp
