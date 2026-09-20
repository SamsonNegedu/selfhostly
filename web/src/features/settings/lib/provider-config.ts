// What is kept for each tunnel provider, keyed by the provider's name.
export type ProviderConfig = Record<string, { api_token?: string; account_id?: string } | undefined>

// A saved token never comes back in full, only with its middle starred out.
const MASK_MARKER = '****'

// Reads the saved provider settings. A token that comes back masked is remembered as a hint and blanked in the
// form, so saving without typing a new one is possible and a mask is never sent back as if it were a token.
// Text that is not valid JSON gives an empty form.
export function parseProviderConfig(raw: string | undefined | null): {
    config: ProviderConfig
    masked: Record<string, string>
} {
    try {
        const parsed: ProviderConfig = raw ? JSON.parse(raw) : {}
        const masked: Record<string, string> = {}
        for (const [name, value] of Object.entries(parsed)) {
            if (value?.api_token?.includes(MASK_MARKER)) {
                masked[name] = value.api_token
                parsed[name] = { ...value, api_token: '' }
            }
        }
        return { config: parsed, masked }
    } catch {
        return { config: {}, masked: {} }
    }
}

// Whether the form has what the provider needs. Cloudflare needs an account ID and a token, and a saved token
// counts. Other providers need nothing typed here.
export function isProviderReady(
    provider: string,
    values: { api_token?: string; account_id?: string },
    masked: Record<string, string>,
): boolean {
    if (provider !== 'cloudflare') return true
    return ((values.api_token ?? '') !== '' || masked[provider] !== undefined) && (values.account_id ?? '') !== ''
}
