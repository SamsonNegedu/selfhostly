import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/Card'
import { Field } from '@/shared/components/ui/Field'
import { Input } from '@/shared/components/ui/Input'
import { RadioCard, RadioGroup } from '@/shared/components/ui/RadioGroup'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/components/ui/Select'
import { Textarea } from '@/shared/components/ui/Textarea'
import type { NewAppForm } from '../hooks/useNewAppForm'
import type { Access } from '../lib/new-app-rules'

// The name, node, port, description and who can reach the app.
function ConfigureCard({ form }: { form: NewAppForm }) {
    const {
        mode,
        name,
        setName,
        nameProblem: error,
        nameTaken: taken,
        nodeId,
        setNodeId,
        nodeName,
        onlineNodes,
        hostPort,
        setHostPort,
        portError,
        portShared,
        description,
        setDescription,
        access,
        setAccess,
        hostname,
        setHostname,
        exposed,
    } = form

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-base">Configure</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-5">
                <div className="grid gap-4 sm:grid-cols-2">
                    <Field
                        label="Name"
                        error={error ?? (taken ? `${nodeName} already has an app called ${name}.` : undefined)}
                        hint="Lowercase letters, numbers and hyphens."
                    >
                        <Input
                            value={name}
                            onChange={(event) => setName(event.target.value)}
                            placeholder="my-app"
                            autoComplete="off"
                        />
                    </Field>
                    <Field label="Node">
                        <Select value={nodeId} onValueChange={setNodeId}>
                            <SelectTrigger aria-label="Node">
                                <SelectValue placeholder="No node is online" />
                            </SelectTrigger>
                            <SelectContent>
                                {onlineNodes.map((node) => (
                                    <SelectItem key={node.id} value={node.id}>
                                        {node.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </Field>
                </div>
                {mode === 'template' && (
                    <Field
                        label="Port on the node"
                        error={portError}
                        hint={portShared ?? 'The address you open it on inside your network.'}
                        className="sm:max-w-[200px]"
                    >
                        <Input
                            value={hostPort}
                            onChange={(event) => setHostPort(event.target.value.replace(/\D/g, ''))}
                            inputMode="numeric"
                            className="font-mono"
                        />
                    </Field>
                )}
                <Field label="Description">
                    <Textarea
                        value={description}
                        onChange={(event) => setDescription(event.target.value)}
                        rows={2}
                        placeholder="Optional"
                    />
                </Field>

                <fieldset className="flex flex-col gap-2">
                    <legend className="mb-1 text-compact font-medium">Who can reach it</legend>
                    <RadioGroup
                        value={access}
                        onValueChange={(value) => setAccess(value as Access)}
                        aria-label="Who can reach it"
                    >
                        <RadioCard
                            value="none"
                            title="Only my network"
                            description="No public address. You can add one later."
                        />
                        <RadioCard
                            value="quick"
                            title="Quick Tunnel"
                            description="A temporary public address on trycloudflare.com. No account needed."
                        />
                        <RadioCard
                            value="custom"
                            title="Your own domain"
                            description="A stable address through Cloudflare. Needs Cloudflare connected in Settings."
                        />
                    </RadioGroup>
                    {access === 'custom' && (
                        <Field
                            label="Hostname"
                            hint={exposed ? `Sends visitors to ${exposed.service} on port ${exposed.port}.` : undefined}
                            className="mt-2 sm:max-w-sm"
                        >
                            <Input
                                value={hostname}
                                onChange={(event) => setHostname(event.target.value)}
                                placeholder="app.example.com"
                                inputMode="url"
                            />
                        </Field>
                    )}
                </fieldset>
            </CardContent>
        </Card>
    )
}

export default ConfigureCard
