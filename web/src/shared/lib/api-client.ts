/**
 * API Client - Centralized HTTP request handling
 * Provides type-safe, consistent API calls with error handling
 */

interface ApiError {
    error: string
    details?: string
}

interface RequestConfig extends RequestInit {
    params?: Record<string, string | number | boolean>
}

/**
 * A failed API call. `code` is stable and machine-readable, and `field` names the request field the error is
 * about, so a form can show it next to the input. `details` is written for people and is safe to show.
 */
export class ApiRequestError extends Error {
    constructor(
        message: string,
        readonly status: number,
        readonly code?: string,
        readonly field?: string,
        readonly details?: string,
    ) {
        super(message)
        this.name = 'ApiRequestError'
    }
}

class ApiClient {
    private baseURL: string

    constructor(baseURL: string = '') {
        this.baseURL = baseURL
    }

    /**
     * Build URL with query parameters
     */
    private buildURL(endpoint: string, params?: Record<string, string | number | boolean>): string {
        const url = `${this.baseURL}${endpoint}`

        if (!params || Object.keys(params).length === 0) {
            return url
        }

        const searchParams = new URLSearchParams()
        Object.entries(params).forEach(([key, value]) => {
            searchParams.append(key, String(value))
        })

        return `${url}?${searchParams.toString()}`
    }

    /**
     * Handle API response and errors
     */
    private async handleResponse<T>(response: Response): Promise<T> {
        // Handle 401 Unauthorized
        if (response.status === 401) {
            throw new Error('UNAUTHORIZED')
        }

        // Handle 404 Not Found (e.g., when auth is disabled). A 404 that carries a code is the API saying a specific
        // thing was not found, so it is reported like any other error.
        if (response.status === 404) {
            const body = await response
                .clone()
                .json()
                .catch(() => null)
            if (body && typeof body === 'object' && 'code' in body) {
                throw new ApiRequestError(
                    body.error || 'Not found',
                    response.status,
                    body.code,
                    body.field,
                    body.details,
                )
            }
            throw new Error('NOT_FOUND')
        }

        // Handle non-OK responses
        if (!response.ok) {
            let errorMessage = `Request failed with status ${response.status}`
            let body: { code?: string; field?: string; details?: string } | null = null

            try {
                const errorData: ApiError & { code?: string; field?: string } = await response.json()
                errorMessage = errorData.error || errorMessage
                if (errorData.details) {
                    errorMessage += `: ${errorData.details}`
                }
                body = errorData
            } catch {
                // If JSON parsing fails, use status text
                errorMessage = response.statusText || errorMessage
            }

            throw new ApiRequestError(errorMessage, response.status, body?.code, body?.field, body?.details)
        }

        // Handle 204 No Content
        if (response.status === 204) {
            return {} as T
        }

        // Parse JSON response
        try {
            return await response.json()
        } catch {
            throw new Error('Failed to parse response JSON')
        }
    }

    /**
     * Make a request with common configuration
     */
    private async request<T>(endpoint: string, config: RequestConfig = {}): Promise<T> {
        const { params, ...fetchConfig } = config

        const url = this.buildURL(endpoint, params)

        const defaultConfig: RequestInit = {
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                ...fetchConfig.headers,
            },
        }

        const finalConfig = { ...defaultConfig, ...fetchConfig }

        const response = await fetch(url, finalConfig)
        return this.handleResponse<T>(response)
    }

    /**
     * GET request
     */
    async get<T>(endpoint: string, params?: Record<string, string | number | boolean>): Promise<T> {
        return this.request<T>(endpoint, {
            method: 'GET',
            params,
        })
    }

    /**
     * POST request
     */
    async post<T, D = unknown>(
        endpoint: string,
        data?: D,
        params?: Record<string, string | number | boolean>,
    ): Promise<T> {
        return this.request<T>(endpoint, {
            method: 'POST',
            body: data ? JSON.stringify(data) : undefined,
            params,
        })
    }

    /**
     * PUT request
     */
    async put<T, D = unknown>(endpoint: string, data?: D): Promise<T> {
        return this.request<T>(endpoint, {
            method: 'PUT',
            body: data ? JSON.stringify(data) : undefined,
        })
    }

    /**
     * PATCH request
     */
    async patch<T, D = unknown>(endpoint: string, data?: D): Promise<T> {
        return this.request<T>(endpoint, {
            method: 'PATCH',
            body: data ? JSON.stringify(data) : undefined,
        })
    }

    /**
     * DELETE request
     */
    async delete<T>(endpoint: string, params?: Record<string, string | number | boolean>): Promise<T> {
        return this.request<T>(endpoint, {
            method: 'DELETE',
            params,
        })
    }
}

// Export singleton instance
export const apiClient = new ApiClient()
