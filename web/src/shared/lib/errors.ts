import { ApiRequestError } from '@/shared/lib/api-client'

const GENERIC_MESSAGE = 'Something went wrong. Try again in a moment.'
const JSON_START = '{'
const NETWORK_MESSAGE = 'Could not reach the server. Check your connection and try again.'
// What browsers report when a request never got an answer.
const NETWORK_ERROR_PATTERN = /failed to fetch|networkerror|load failed|network request failed/i

interface ApiErrorBody {
    error?: string
    details?: string
    message?: string
}

// Errors can come from another frame or realm, where `instanceof Error` fails, so look for a message instead.
function messageOf(error: unknown): string {
    if (typeof error === 'string') return error
    if (typeof error === 'object' && error !== null && 'message' in error) {
        const { message } = error as { message: unknown }
        return typeof message === 'string' ? message : ''
    }
    return ''
}

// Turns whatever was thrown into a sentence a person can read. API failures often arrive as a raw
// JSON body such as {"error":"...","details":"..."}, which must never be shown as is.
export function describeError(error: unknown): string {
    // An error with a code has a sentence written for people, so it is shown as is.
    if (error instanceof ApiRequestError && error.code && error.details) return error.details

    const raw = messageOf(error)
    const text = raw.trim()
    if (!text) return GENERIC_MESSAGE

    if (NETWORK_ERROR_PATTERN.test(text)) return NETWORK_MESSAGE

    if (text.startsWith(JSON_START)) {
        try {
            const body = JSON.parse(text) as ApiErrorBody
            return body.error ?? body.message ?? GENERIC_MESSAGE
        } catch {
            return GENERIC_MESSAGE
        }
    }

    return text
}

// The message for one form field, when the server said the error is about that field.
export function fieldError(error: unknown, field: string): string | undefined {
    return error instanceof ApiRequestError && error.field === field ? describeError(error) : undefined
}
