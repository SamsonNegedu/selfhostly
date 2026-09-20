// Lets Node load the app's TypeScript files, which import each other without a file extension.
export async function resolve(specifier, context, nextResolve) {
    try {
        return await nextResolve(specifier, context)
    } catch (error) {
        if (error.code === 'ERR_MODULE_NOT_FOUND' && /^\.\.?\//.test(specifier)) {
            for (const suffix of ['.ts', '/index.ts']) {
                try {
                    return await nextResolve(specifier + suffix, context)
                } catch {
                    // Try the next spelling.
                }
            }
        }
        throw error
    }
}
