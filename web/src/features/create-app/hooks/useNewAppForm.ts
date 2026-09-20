import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/shared/components/ui/Toast'
import { describeError } from '@/shared/lib/errors'
import { appHref } from '@/shared/lib/routes'
import { useApps, useCreateApp, useNodes } from '@/shared/services/api'
import { checkCompose } from '@/features/app-details/lib/compose-checks'
import { collectHostPorts, suggestFreePort } from '@/features/app-details/lib/port-conflict'
import { firstExposedService } from '../lib/compose-access'
import {
    blockedReason,
    buildChecks,
    nameError,
    portError as portErrorFor,
    portSharedMessage,
    type Access,
    type Mode,
} from '../lib/new-app-rules'
import { TEMPLATES, templateCompose, toRawUrl, type AppTemplate } from '../lib/templates'

export type FetchState =
    | { status: 'idle' }
    | { status: 'loading' }
    | { status: 'ok' }
    | { status: 'error'; message: string }

export type NewAppForm = ReturnType<typeof useNewAppForm>

// Everything the New app page knows: what was chosen, what that produces (the compose file, the checks, why Create
// is not available yet), and the actions. The page and its sections only draw it.
export function useNewAppForm() {
    const navigate = useNavigate()
    const { toast } = useToast()
    const createApp = useCreateApp()
    const { data: nodes = [] } = useNodes()
    const { data: apps = [] } = useApps(undefined)

    const [mode, setMode] = useState<Mode>('template')
    const [templateId, setTemplateId] = useState<string | null>(null)
    const [hostPort, setHostPort] = useState('')
    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [nodeId, setNodeId] = useState('')
    const [pasted, setPasted] = useState('')
    const [link, setLink] = useState('')
    const [fetched, setFetched] = useState<FetchState>({ status: 'idle' })
    const [access, setAccess] = useState<Access>('none')
    const [hostname, setHostname] = useState('')

    const onlineNodes = nodes.filter((node) => node.status === 'online')
    useEffect(() => {
        if (!nodeId && onlineNodes.length > 0)
            setNodeId((onlineNodes.find((node) => node.is_primary) ?? onlineNodes[0]).id)
    }, [nodeId, onlineNodes])

    const template = TEMPLATES.find((item) => item.id === templateId) ?? null
    const nodeApps = useMemo(() => apps.filter((app) => app.node_id === nodeId), [apps, nodeId])
    const usedPorts = useMemo(() => collectHostPorts(nodeApps.map((app) => app.compose_content)), [nodeApps])

    // A template starts on its usual port, or the next one no other app on the node publishes.
    useEffect(() => {
        if (!template) return
        const free = suggestFreePort(template.defaultHostPort - 1, usedPorts)
        setHostPort(String(free ?? template.defaultHostPort))
        // Only the template and the node choose the starting port. Typing in the field must not reset it.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [templateId, nodeId, apps.length])

    const port = Number(hostPort)
    const content =
        mode === 'template'
            ? template
                ? templateCompose(template, name, Number.isInteger(port) && port > 0 ? port : template.defaultHostPort)
                : ''
            : pasted
    const exposed = useMemo(() => firstExposedService(content), [content])
    const composeResult = useMemo(
        () => (content.trim() === '' ? null : checkCompose(content, nodeApps)),
        [content, nodeApps],
    )

    const nameProblem = nameError(name)
    const nameTaken = name !== '' && !nameProblem && nodeApps.some((app) => app.name === name)
    const nodeName = nodes.find((node) => node.id === nodeId)?.name ?? nodeId
    const portError = portErrorFor(mode, port)
    const portShared = portSharedMessage(mode, port, !!portError, usedPorts, nodeName)

    const rules = {
        mode,
        hasTemplate: template !== null,
        content,
        name,
        nameProblem,
        nameTaken,
        nodeId,
        nodeName,
        portError,
        portShared,
        composeChecks: composeResult?.checks ?? [],
        access,
        hasExposedPort: !!exposed,
        hostname,
    }
    const checks = buildChecks(rules)
    const blocked = blockedReason(rules, checks)

    // Choosing a template also fills in the name, unless the person has already typed their own.
    const selectTemplate = (item: AppTemplate) => {
        setTemplateId(item.id)
        if (name === '' || TEMPLATES.some((other) => other.id === name)) setName(item.id)
    }

    const fetchLink = async () => {
        setFetched({ status: 'loading' })
        try {
            const response = await fetch(toRawUrl(link))
            if (!response.ok) throw new Error(`The address answered ${response.status}.`)
            const text = await response.text()
            if (text.trim() === '') throw new Error('The file is empty.')
            setPasted(text)
            setFetched({ status: 'ok' })
        } catch (failure) {
            const network = failure instanceof TypeError
            setFetched({
                status: 'error',
                message: network
                    ? 'Could not reach that address. Check the link, or paste the file instead.'
                    : describeError(failure),
            })
        }
    }

    const create = () => {
        if (blocked) return
        createApp.mutate(
            {
                name,
                description,
                compose_content: content,
                node_id: nodeId,
                tunnel_mode: access === 'none' ? undefined : access,
                quick_tunnel_service: access === 'quick' ? exposed?.service : undefined,
                quick_tunnel_port: access === 'quick' ? exposed?.port : undefined,
                ingress_rules:
                    access === 'custom' && exposed
                        ? [
                              {
                                  hostname: hostname.trim(),
                                  service: `http://${exposed.service}:${exposed.port}`,
                                  path: null,
                              },
                          ]
                        : undefined,
            },
            {
                onSuccess: (created) => {
                    toast.success('App created', `${created.name} is being deployed`)
                    navigate(appHref(created, 'logs'))
                },
            },
        )
    }

    const showConfigure = mode === 'template' ? template !== null : content.trim() !== '' || mode === 'paste'

    return {
        mode,
        setMode,
        template,
        templateId,
        selectTemplate,
        name,
        setName,
        nameProblem,
        nameTaken,
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
        pasted,
        setPasted,
        link,
        setLink,
        fetched,
        fetchLink,
        content,
        checks,
        blocked,
        showConfigure,
        create,
        creating: createApp.isPending,
        createError: createApp.error,
    }
}
