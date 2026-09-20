import { apiClient } from '../../lib/api-client'
import type { UpdateStatus, PlanUpdateRequest, ApplyUpdateRequest } from '../../types/api'

// ============================================================================
// UI-driven updates
// ============================================================================

const UPDATE_PATH = '/api/system/update'

// These routes act on the primary itself, so none of them takes a node_id. The write calls answer 202 and the
// outcome is read back through `status`.
export const updateApi = {
    status: () => apiClient.get<UpdateStatus>(UPDATE_PATH),
    check: () => apiClient.post<UpdateStatus>(`${UPDATE_PATH}/check`),
    plan: (request: PlanUpdateRequest) =>
        apiClient.post<UpdateStatus, PlanUpdateRequest>(`${UPDATE_PATH}/plan`, request),
    apply: (request: ApplyUpdateRequest) =>
        apiClient.post<UpdateStatus, ApplyUpdateRequest>(`${UPDATE_PATH}/apply`, request),
    rollback: () => apiClient.post<UpdateStatus>(`${UPDATE_PATH}/rollback`),
}
