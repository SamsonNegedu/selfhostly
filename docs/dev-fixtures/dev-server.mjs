// Starts the web app's Vite config with a polling file watcher.
// Some sandboxes miss file-system events, so Vite keeps serving stale modules after an edit.
// Usage from the repo root: node docs/dev-fixtures/dev-server.mjs
import { pathToFileURL } from 'node:url'

const POLL_INTERVAL_MS = 300
// Unusual ports, so this never collides with a developer's own stack on 8080 or 5173.
const WEB_PORT = 5183
const API_URL = 'http://localhost:8090'
const webRoot = new URL('../../web/', import.meta.url)
const root = webRoot.pathname.replace(/\/$/, '')
const viteEntry = pathToFileURL(`${root}/node_modules/vite/dist/node/index.js`).href
const { createServer, loadConfigFromFile, mergeConfig } = await import(viteEntry)

process.chdir(root)
process.env.VITE_API_BASE ??= API_URL

const loaded = await loadConfigFromFile({ command: 'serve', mode: 'development' }, `${root}/vite.config.ts`, root)
const server = await createServer(
  mergeConfig(loaded.config, {
    configFile: false,
    root,
    server: { host: 'localhost', port: WEB_PORT, strictPort: true, watch: { usePolling: true, interval: POLL_INTERVAL_MS } },
  }),
)
await server.listen()
server.printUrls()
