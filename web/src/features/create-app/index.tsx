import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCreateApp } from '@/shared/services/api'
import { Button } from '@/shared/components/ui'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import ComposeEditor from './components/ComposeEditor'
import PreviewCompose from './components/PreviewCompose'
import ProgressIndicator from './components/ProgressIndicator'
import ConfigurationChecklist from './components/ConfigurationChecklist'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'
import { ArrowRight, ArrowLeft, Sparkles, Shield, CheckCircle2 } from 'lucide-react'

type StepType = 'information' | 'compose' | 'review'

function CreateApp() {
    const navigate = useNavigate()
    const createApp = useCreateApp()
    const [currentStep, setCurrentStep] = useState<StepType>('information')
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [touched, setTouched] = useState<Record<string, boolean>>({})

    const [formData, setFormData] = useState({
        name: '',
        description: '',
        compose_content: '',
    })

    const steps = [
        { id: 1, label: 'Information', status: currentStep === 'information' ? 'current' as const : (['compose', 'review'].includes(currentStep) ? 'completed' as const : 'pending' as const) },
        { id: 2, label: 'Compose', status: currentStep === 'compose' ? 'current' as const : currentStep === 'review' ? 'completed' as const : 'pending' as const },
        { id: 3, label: 'Review', status: currentStep === 'review' ? 'current' as const : 'pending' as const },
    ]

    // Form validation
    const validateField = (name: string, value: string): string | null => {
        switch (name) {
            case 'name':
                if (!value.trim()) return 'App name is required'
                if (!/^[a-z0-9-]+$/.test(value)) return 'Only lowercase letters, numbers, and hyphens allowed'
                if (value.length < 3) return 'Name must be at least 3 characters'
                if (value.length > 63) return 'Name must be less than 64 characters'
                return null
            case 'compose_content':
                if (!value.trim()) return 'Docker Compose configuration is required'
                return null
            default:
                return null
        }
    }

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {}

        if (validateField('name', formData.name)) {
            newErrors.name = validateField('name', formData.name)!
        }
        if (validateField('compose_content', formData.compose_content)) {
            newErrors.compose_content = validateField('compose_content', formData.compose_content)!
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleFieldChange = (name: string, value: string) => {
        setFormData({ ...formData, [name]: value })

        if (touched[name]) {
            const error = validateField(name, value)
            setErrors(prev => ({
                ...prev,
                [name]: error || ''
            }))
        }
    }

    const handleFieldBlur = (name: string) => {
        setTouched(prev => ({ ...prev, [name]: true }))
        const error = validateField(name, formData[name as keyof typeof formData])
        setErrors(prev => ({
            ...prev,
            [name]: error || ''
        }))
    }

    const handleNext = () => {
        if (currentStep === 'information') {
            const error = validateField('name', formData.name)
            if (error) {
                setErrors({ name: error })
                setTouched({ name: true })
                return
            }
            setCurrentStep('compose')
        } else if (currentStep === 'compose') {
            if (!validateForm()) return
            setCurrentStep('review')
        }
    }

    const handleBack = () => {
        if (currentStep === 'compose') setCurrentStep('information')
        else if (currentStep === 'review') setCurrentStep('compose')
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!validateForm()) return

        createApp.mutate(formData, {
            onSuccess: () => {
                navigate('/dashboard')
            },
        })
    }

    // Configuration checklist for review step
    const checklist = [
        {
            id: '1',
            label: 'App name provided',
            checked: !!formData.name && !errors.name
        },
        {
            id: '2',
            label: 'Docker Compose configured',
            checked: !!formData.compose_content && !errors.compose_content
        },
        {
            id: '3',
            label: 'Cloudflare Tunnel will be configured',
            checked: true
        }
    ]

    const canProceed = currentStep === 'information'
        ? !!formData.name && !errors.name
        : currentStep === 'compose'
            ? !!formData.compose_content && !errors.compose_content
            : true

    return (
        <div className="fade-in">
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

            {/* Progress Indicator */}
            <ProgressIndicator steps={steps} />

            <div className="max-w-3xl">
                {/* Step 1: App Information */}
                {currentStep === 'information' && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Sparkles className="h-5 w-5 text-primary" />
                                App Information
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div>
                                <label htmlFor="name" className="block text-sm font-medium mb-2">
                                    App Name <span className="text-destructive">*</span>
                                </label>
                                <input
                                    id="name"
                                    type="text"
                                    value={formData.name}
                                    onChange={(e) => handleFieldChange('name', e.target.value)}
                                    onBlur={() => handleFieldBlur('name')}
                                    className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-ring bg-background transition-colors ${errors.name ? 'border-red-500 focus:ring-red-500' : ''}`}
                                    placeholder="my-awesome-app"
                                />
                                {errors.name && touched.name && (
                                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.name}</p>
                                )}
                                <p className="text-xs text-muted-foreground mt-1">
                                    Use lowercase letters, numbers, and hyphens only. Max 63 characters.
                                </p>
                            </div>

                            <div>
                                <label htmlFor="description" className="block text-sm font-medium mb-2">
                                    Description
                                </label>
                                <textarea
                                    id="description"
                                    value={formData.description}
                                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                    className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-ring min-h-[100px] bg-background resize-y"
                                    placeholder="A brief description of your app (optional)"
                                />
                            </div>

                            <div className="flex justify-end">
                                <Button
                                    onClick={handleNext}
                                    disabled={!canProceed}
                                    className="button-press"
                                >
                                    Next Step
                                    <ArrowRight className="h-4 w-4 ml-2" />
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                )}

                {/* Step 2: Docker Compose */}
                {currentStep === 'compose' && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Shield className="h-5 w-5 text-primary" />
                                Docker Compose Configuration
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div>
                                <label className="block text-sm font-medium mb-2">
                                    Compose File <span className="text-destructive">*</span>
                                </label>
                                <ComposeEditor
                                    value={formData.compose_content}
                                    onChange={(value) => handleFieldChange('compose_content', value)}
                                />
                                {errors.compose_content && (
                                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.compose_content}</p>
                                )}
                            </div>

                            <div className="flex justify-between">
                                <Button
                                    variant="outline"
                                    onClick={handleBack}
                                    className="button-press"
                                >
                                    <ArrowLeft className="h-4 w-4 mr-2" />
                                    Back
                                </Button>
                                <Button
                                    onClick={handleNext}
                                    disabled={!canProceed}
                                    className="button-press"
                                >
                                    Review
                                    <ArrowRight className="h-4 w-4 ml-2" />
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                )}

                {/* Step 3: Review & Deploy */}
                {currentStep === 'review' && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <CheckCircle2 className="h-5 w-5 text-primary" />
                                Review & Deploy
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            {/* Configuration Checklist */}
                            <ConfigurationChecklist items={checklist} />

                            {/* Summary */}
                            <div className="space-y-4">
                                <div className="p-4 bg-muted/50 rounded-lg">
                                    <h3 className="font-semibold mb-3">Summary</h3>
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                                        <div>
                                            <span className="text-muted-foreground">Name:</span>
                                            <p className="font-medium mt-1">{formData.name}</p>
                                        </div>
                                        <div>
                                            <span className="text-muted-foreground">Description:</span>
                                            <p className="font-medium mt-1">{formData.description || 'No description'}</p>
                                        </div>
                                    </div>
                                </div>

                                <div>
                                    <h3 className="font-semibold mb-3">Docker Compose Preview</h3>
                                    <PreviewCompose content={formData.compose_content} />
                                </div>
                            </div>

                            {createApp.error && (
                                <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                    <p className="text-sm text-red-600 dark:text-red-400">
                                        {createApp.error.message}
                                    </p>
                                </div>
                            )}

                            <div className="flex justify-between">
                                <Button
                                    variant="outline"
                                    onClick={handleBack}
                                    disabled={createApp.isPending}
                                    className="button-press"
                                >
                                    <ArrowLeft className="h-4 w-4 mr-2" />
                                    Back
                                </Button>
                                <Button
                                    onClick={handleSubmit}
                                    disabled={createApp.isPending || !checklist.every(i => i.checked)}
                                    className="button-press"
                                >
                                    {createApp.isPending ? (
                                        <>
                                            <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                            Deploying...
                                        </>
                                    ) : (
                                        <>
                                            <Sparkles className="h-4 w-4 mr-2" />
                                            Deploy App
                                        </>
                                    )}
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                )}
            </div>
        </div>
    )
}

export default CreateApp
