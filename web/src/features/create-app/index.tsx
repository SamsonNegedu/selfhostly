import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCreateApp } from '@/shared/services/api'
import { Button } from '@/shared/components/ui'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { NodeSelector } from '@/shared/components/ui/NodeSelector'
import ComposeEditor from './components/ComposeEditor'
import PreviewCompose from './components/PreviewCompose'
import ProgressIndicator from './components/ProgressIndicator'
import ConfigurationChecklist from './components/ConfigurationChecklist'
import IngressRulesEditor from './components/IngressRulesEditor'
import AppBreadcrumb from '@/shared/components/layout/Breadcrumb'
import { ArrowRight, ArrowLeft, Sparkles, Shield, CheckCircle2, Globe } from 'lucide-react'
import type { IngressRule } from '@/shared/types/api'

type StepType = 'information' | 'compose' | 'ingress' | 'review'

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
        ingress_rules: [] as IngressRule[],
        node_id: '', // Target node for deployment (empty = current node)
        tunnel_mode: '' as '' | 'custom' | 'quick',
        quick_tunnel_service: '',
        quick_tunnel_port: 80,
    })

    const showIngressStep = formData.tunnel_mode === 'custom'
    const STEP_LABELS: Record<StepType, string> = {
        information: 'Information',
        compose: 'Compose',
        ingress: 'Ingress (Optional)',
        review: 'Review',
    }
    const stepOrder: StepType[] = showIngressStep
        ? ['information', 'compose', 'ingress', 'review']
        : ['information', 'compose', 'review']
    const currentIndex = stepOrder.includes(currentStep)
        ? stepOrder.indexOf(currentStep)
        : stepOrder.length - 1
    const steps = stepOrder.map((stepKey, i) => ({
        id: i + 1,
        label: STEP_LABELS[stepKey],
        status: (i < currentIndex ? 'completed' : i === currentIndex ? 'current' : 'pending') as 'pending' | 'current' | 'completed',
    }))

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

    const handleFieldChange = (name: string, value: string | number | IngressRule[] | '' | 'custom' | 'quick') => {
        setFormData(prev => ({ ...prev, [name]: value }))

        if (touched[name] && (typeof value === 'string' || typeof value === 'number')) {
            const error = typeof value === 'string' ? validateField(name, value) : null
            setErrors(prev => ({
                ...prev,
                [name]: error || ''
            }))
        }
    }

    const handleFieldBlur = (name: string) => {
        setTouched(prev => ({ ...prev, [name]: true }))
        const value = formData[name as keyof typeof formData]
        if (typeof value === 'string') {
            const error = validateField(name, value)
            setErrors(prev => ({
                ...prev,
                [name]: error || ''
            }))
        }
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
            setCurrentStep(showIngressStep ? 'ingress' : 'review')
        } else if (currentStep === 'ingress') {
            setCurrentStep('review')
        }
    }

    const handleSkipIngress = () => {
        setFormData({ ...formData, ingress_rules: [] })
        setCurrentStep('review')
    }

    const handleBack = () => {
        if (currentStep === 'compose') setCurrentStep('information')
        else if (currentStep === 'ingress') setCurrentStep('compose')
        else if (currentStep === 'review') setCurrentStep(formData.tunnel_mode === 'custom' ? 'ingress' : 'compose')
    }

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!validateForm()) return

        // Filter out empty ingress rules
        const validIngressRules = formData.ingress_rules.filter(rule => rule.service.trim() !== '')

        const submitData = {
            name: formData.name,
            description: formData.description,
            compose_content: formData.compose_content,
            ingress_rules: validIngressRules.length > 0 ? validIngressRules : undefined,
            node_id: formData.node_id || undefined,
            tunnel_mode: formData.tunnel_mode || undefined,
            quick_tunnel_service: formData.tunnel_mode === 'quick' ? formData.quick_tunnel_service.trim() : undefined,
            quick_tunnel_port: formData.tunnel_mode === 'quick' ? formData.quick_tunnel_port : undefined,
        }

        createApp.mutate(submitData, {
            onSuccess: (data) => {
                // Redirect to the newly created app's details page
                navigate(`/apps/${data.id}`)
            },
        })
    }

    // Configuration checklist for review step (depends on tunnel mode)
    const hasValidIngressRules = formData.ingress_rules.some(rule => rule.service.trim() !== '')
    const baseChecklist = [
        { id: '1', label: 'App name provided', checked: !!formData.name && !errors.name },
        { id: '2', label: 'Docker Compose configured', checked: !!formData.compose_content && !errors.compose_content },
    ]
    const tunnelChecklistItem =
        formData.tunnel_mode === 'custom'
            ? { id: '3', label: hasValidIngressRules ? 'Ingress rules configured' : 'Ingress rules will use default', checked: true }
            : formData.tunnel_mode === 'quick'
                ? { id: '3', label: `Quick Tunnel: ${formData.quick_tunnel_service || '?'}:${formData.quick_tunnel_port}`, checked: !!formData.quick_tunnel_service && formData.quick_tunnel_port >= 1 }
                : { id: '3', label: 'No tunnel', checked: true }
    const checklist = [...baseChecklist, tunnelChecklistItem]

    const canProceed = currentStep === 'information'
        ? !!formData.name && !errors.name
        : currentStep === 'compose'
            ? !!formData.compose_content && !errors.compose_content
            : currentStep === 'ingress'
                ? true // Ingress is optional
                : true

    return (
        <div className="fade-in">
            {/* Breadcrumb Navigation - Desktop only */}
            <AppBreadcrumb
                items={[
                    { label: 'Home', path: '/apps' },
                    { label: 'Apps', path: '/apps' },
                    { label: 'New App', isCurrentPage: true }
                ]}
                className="mb-4 sm:mb-6"
            />

            <div className="mb-4 sm:mb-6">
                <h1 className="text-2xl sm:text-3xl font-bold">Create New App</h1>
                <p className="text-muted-foreground mt-1 sm:mt-2 text-sm sm:text-base">
                    Deploy a new self-hosted application
                </p>
            </div>

            {/* Progress Indicator */}
            <ProgressIndicator steps={steps} />

            <div className={currentStep === 'review' ? 'w-full' : 'max-w-3xl'}>
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
                                    className={`flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 transition-colors ${errors.name ? 'border-destructive focus:ring-destructive' : ''}`}
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
                                    className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 resize-y"
                                    placeholder="A brief description of your app (optional)"
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">
                                    Tunnel
                                </label>
                                <select
                                    value={formData.tunnel_mode}
                                    onChange={(e) => handleFieldChange('tunnel_mode', e.target.value as '' | 'custom' | 'quick')}
                                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                >
                                    <option value="">No tunnel</option>
                                    <option value="custom">Custom domain (requires Cloudflare credentials)</option>
                                    <option value="quick">Quick Tunnel (temporary trycloudflare.com URL)</option>
                                </select>
                                {formData.tunnel_mode === 'quick' && (
                                    <div className="mt-4 p-4 rounded-lg border border-muted bg-muted/30 space-y-4">
                                        <p className="text-sm text-muted-foreground">
                                            Quick Tunnels are temporary and limited to 200 concurrent requests. No credentials required.
                                        </p>
                                        <div>
                                            <label htmlFor="quick_tunnel_service" className="block text-sm font-medium mb-1">
                                                Target service name <span className="text-destructive">*</span>
                                            </label>
                                            <input
                                                id="quick_tunnel_service"
                                                type="text"
                                                value={formData.quick_tunnel_service}
                                                onChange={(e) => handleFieldChange('quick_tunnel_service', e.target.value)}
                                                onBlur={() => setTouched(prev => ({ ...prev, quick_tunnel_service: true }))}
                                                className={`flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ${errors.quick_tunnel_service ? 'border-destructive' : ''}`}
                                                placeholder="web"
                                            />
                                            {errors.quick_tunnel_service && (
                                                <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.quick_tunnel_service}</p>
                                            )}
                                            <p className="text-xs text-muted-foreground mt-1">The service name from your docker-compose to expose.</p>
                                        </div>
                                        <div>
                                            <label htmlFor="quick_tunnel_port" className="block text-sm font-medium mb-1">
                                                Target port <span className="text-destructive">*</span>
                                            </label>
                                            <input
                                                id="quick_tunnel_port"
                                                type="number"
                                                min={1}
                                                max={65535}
                                                value={formData.quick_tunnel_port}
                                                onChange={(e) => handleFieldChange('quick_tunnel_port', parseInt(e.target.value, 10) || 80)}
                                                className={`flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ${errors.quick_tunnel_port ? 'border-destructive' : ''}`}
                                            />
                                            {errors.quick_tunnel_port && (
                                                <p className="text-sm text-red-600 dark:text-red-400 mt-1">{errors.quick_tunnel_port}</p>
                                            )}
                                            <p className="text-xs text-muted-foreground mt-1">The port your service listens on (1–65535).</p>
                                        </div>
                                    </div>
                                )}
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">
                                    Deployment Node
                                </label>
                                <NodeSelector
                                    selectedNodeIds={formData.node_id ? [formData.node_id] : []}
                                    onChange={(nodeIds) => setFormData(prev => ({ ...prev, node_id: nodeIds[0] || '' }))}
                                    multiSelect={false}
                                />
                                <p className="text-xs text-muted-foreground mt-1">
                                    Choose which node will host this app. Leave unselected to use current node.
                                </p>
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

                {/* Step 3: Ingress Configuration (Optional) */}
                {currentStep === 'ingress' && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Globe className="h-5 w-5 text-primary" />
                                Ingress Configuration (Optional)
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="text-sm text-muted-foreground">
                                Configure how your app will be accessible via Cloudflare Tunnel.
                                You can skip this step and configure it later from the app details page.
                            </div>

                            <IngressRulesEditor
                                value={formData.ingress_rules}
                                onChange={(rules) => setFormData({ ...formData, ingress_rules: rules })}
                            />

                            <div className="flex justify-between">
                                <Button
                                    variant="outline"
                                    onClick={handleBack}
                                    className="button-press"
                                >
                                    <ArrowLeft className="h-4 w-4 mr-2" />
                                    Back
                                </Button>
                                <div className="flex gap-2">
                                    <Button
                                        variant="ghost"
                                        onClick={handleSkipIngress}
                                        className="button-press"
                                    >
                                        Skip for Now
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
                            </div>
                        </CardContent>
                    </Card>
                )}

                {/* Step 4: Review & Deploy */}
                {currentStep === 'review' && (
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        {/* Left Pane - Configuration Details */}
                        <Card className="h-fit">
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
                                        <div className="space-y-3 text-sm">
                                            <div>
                                                <span className="text-muted-foreground">Name:</span>
                                                <p className="font-medium mt-1">{formData.name}</p>
                                            </div>
                                            {formData.description && (
                                                <div>
                                                    <span className="text-muted-foreground">Description:</span>
                                                    <p className="font-medium mt-1">{formData.description}</p>
                                                </div>
                                            )}
                                        </div>
                                    </div>

                                    {showIngressStep && hasValidIngressRules && (
                                        <div>
                                            <h3 className="font-semibold mb-3">Ingress Rules</h3>
                                            <div className="p-4 bg-muted/50 rounded-lg space-y-2">
                                                {formData.ingress_rules
                                                    .filter(rule => rule.service.trim() !== '')
                                                    .map((rule, index) => (
                                                        <div key={index} className="flex items-center gap-2 text-sm">
                                                            <Globe className="h-4 w-4 text-primary flex-shrink-0" />
                                                            <span className="font-medium truncate">
                                                                {rule.hostname || 'Default tunnel URL'}
                                                            </span>
                                                            <span className="text-muted-foreground">→</span>
                                                            <span className="text-muted-foreground truncate">{rule.service}</span>
                                                            {rule.path && (
                                                                <span className="text-muted-foreground text-xs">({rule.path})</span>
                                                            )}
                                                        </div>
                                                    ))}
                                            </div>
                                        </div>
                                    )}
                                </div>

                                {createApp.error && (
                                    <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                        <p className="text-sm text-red-600 dark:text-red-400">
                                            {createApp.error.message}
                                        </p>
                                    </div>
                                )}

                                <div className="flex justify-between pt-4">
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

                        {/* Right Pane - Docker Compose Preview */}
                        <Card className="h-fit lg:sticky lg:top-6">
                            <CardHeader>
                                <CardTitle className="text-lg">Docker Compose Preview</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <PreviewCompose content={formData.compose_content} height="500px" />
                            </CardContent>
                        </Card>
                    </div>
                )}
            </div>
        </div>
    )
}

export default CreateApp
