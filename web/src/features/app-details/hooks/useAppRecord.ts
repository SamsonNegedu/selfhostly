import { useEffect, useMemo, useRef } from 'react'
import { useApp, useApps } from '@/shared/services/api'
import { useAppStore } from '@/shared/stores/app-store'

// An app's record. Which node it lives on is needed to ask for it, and comes from the address if it has one, then
// from the apps already loaded, and last from the list of every node's apps. `isLoading` covers that lookup too.
export function useAppRecord(appId: string | undefined, nodeIdFromUrl: string | null) {
    // Get node_id from app store if available (for initial load)
    const apps = useAppStore((state) => state.apps)
    const cachedApp = apps.find((a) => a.id === appId)

    // Fetch apps list if nodeId is not available from cache or URL
    const shouldFetchApps = !nodeIdFromUrl && !cachedApp?.node_id
    const { data: appsList, isLoading: isLoadingApps } = useApps(undefined) // Fetch from all nodes

    // Determine nodeId: URL param > cache > fetched apps list
    const nodeId = useMemo(() => {
        if (nodeIdFromUrl) return nodeIdFromUrl
        if (cachedApp?.node_id) return cachedApp.node_id
        if (appsList) {
            const foundApp = appsList.find((a) => a.id === appId)
            return foundApp?.node_id
        }
        return undefined
    }, [nodeIdFromUrl, cachedApp?.node_id, appsList, appId])

    const { data: app, isLoading: isLoadingApp, refetch, isFetching } = useApp(appId!, nodeId || '')
    // Track if nodeId was previously undefined (query was disabled)
    const prevNodeIdRef = useRef<string | undefined>(undefined)

    // Refetch app data when nodeId becomes available for the first time
    // This ensures we get fresh data from the backend, not stale cached data
    useEffect(() => {
        const wasDisabled = prevNodeIdRef.current === undefined || prevNodeIdRef.current === ''
        const isNowEnabled = !!nodeId && nodeId !== ''

        if (wasDisabled && isNowEnabled && appId) {
            // Query was just enabled - ensure we refetch to get deterministic backend state
            // This handles the case where app was created and we navigated here before nodeId was resolved
            refetch()
        }

        prevNodeIdRef.current = nodeId
    }, [appId, nodeId, refetch])

    // Wait for the apps list too, if it is needed to find the node.
    const isLoading = isLoadingApp || (shouldFetchApps && isLoadingApps)
    return { app, isLoading, refetch, isFetching }
}
