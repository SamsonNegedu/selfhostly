import { useEffect, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiRequestError } from '@/shared/lib/api-client'
import { updateApi } from '@/shared/services/api'
import { isRunActive, isRunFinished, isUpdateBusy } from '@/shared/lib/update'
import type { ApplyUpdateRequest, PlanUpdateRequest, UpdateStatus } from '@/shared/types/api'

export const UPDATE_QUERY_KEY = ['update'] as const

const POLL_ACTIVE_MS = 2_000
const POLL_IDLE_MS = 5 * 60 * 1_000
// Remembers which run or version this tab already reloaded for, so a reload can never repeat.
const RELOADED_FOR_KEY = 'selfhostly:update-reloaded-for'

// The update state, polled quickly while a review or a run is under way and rarely otherwise. When a request
// fails while something is under way, the API is most likely restarting: the last known state stays and
// `reconnecting` is true until it answers again.
export function useUpdateStatus() {
    const query = useQuery<UpdateStatus>({
        queryKey: UPDATE_QUERY_KEY,
        queryFn: updateApi.status,
        refetchInterval: (current) => (isUpdateBusy(current.state.data) ? POLL_ACTIVE_MS : POLL_IDLE_MS),
        // The update itself takes the API down for a moment, so keep asking while the tab is in the background.
        refetchIntervalInBackground: true,
        // The interval is the retry: a failed request is asked again on the next tick.
        retry: false,
    })
    const busy = isUpdateBusy(query.data)
    return { ...query, busy, reconnecting: busy && query.isError }
}

function isUpdateStatus(value: unknown): value is UpdateStatus {
    return typeof value === 'object' && value !== null && 'current_version' in value
}

function useUpdateMutation<Variables = void>(call: (variables: Variables) => Promise<UpdateStatus>) {
    const queryClient = useQueryClient()
    return useMutation<UpdateStatus, Error, Variables>({
        mutationFn: call,
        onSuccess: (data) => {
            if (isUpdateStatus(data)) queryClient.setQueryData(UPDATE_QUERY_KEY, data)
            // The write calls only start the work, so ask again for what the server now says.
            void queryClient.invalidateQueries({ queryKey: UPDATE_QUERY_KEY })
        },
        onError: (error) => {
            // The server's idea of the available release only lives in memory: a restart, or another check
            // finding something else, can make the version this tab still shows unknown to it. Refreshing
            // now means the stale card updates (or goes away) instead of just failing the same way again.
            if (error instanceof ApiRequestError && error.code === 'unknown_version') {
                void queryClient.invalidateQueries({ queryKey: UPDATE_QUERY_KEY })
            }
        },
    })
}

export const useCheckUpdate = () => useUpdateMutation<void>(() => updateApi.check())
export const usePlanUpdate = () => useUpdateMutation<PlanUpdateRequest>(updateApi.plan)
export const useApplyUpdate = () => useUpdateMutation<ApplyUpdateRequest>(updateApi.apply)
export const useRollbackUpdate = () => useUpdateMutation<void>(() => updateApi.rollback())

function reloadOnce(key: string) {
    try {
        if (sessionStorage.getItem(RELOADED_FOR_KEY) === key) return
        sessionStorage.setItem(RELOADED_FOR_KEY, key)
    } catch {
        // Without storage the guard cannot be kept, and a reload that could repeat is worse than a stale page.
        return
    }
    window.location.reload()
}

// The web interface is replaced by an update, so a page opened before it may ask for files that no longer
// exist. When a run this tab watched has ended, or the server reports another version than this tab first saw,
// refresh the data and reload once.
export function useReloadAfterUpdate() {
    const queryClient = useQueryClient()
    const { data } = useUpdateStatus()
    const firstVersion = useRef<string | null>(null)
    const watchedRun = useRef<string | null>(null)

    useEffect(() => {
        if (!data) return
        firstVersion.current ??= data.current_version
        const run = data.run
        if (run && isRunActive(run)) {
            watchedRun.current = run.id
            return
        }
        const finishedWhileWatching = !!run && isRunFinished(run) && watchedRun.current === run.id
        const versionChanged = data.current_version !== firstVersion.current
        if (!finishedWhileWatching && !versionChanged) return
        void queryClient.invalidateQueries()
        reloadOnce(finishedWhileWatching && run ? run.id : data.current_version)
    }, [data, queryClient])
}
