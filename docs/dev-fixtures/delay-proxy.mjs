// Adds artificial latency in front of the backend so loading states can be observed.
// Usage: DELAY_MS=2500 node docs/dev-fixtures/delay-proxy.mjs
// Then start the frontend with VITE_API_BASE=http://localhost:8091
import http from 'node:http'

const LISTEN_PORT = 8091
const BACKEND_PORT = 8090
const DELAY_MS = Number(process.env.DELAY_MS ?? 2000)

http
  .createServer((req, res) => {
    setTimeout(() => {
      const upstream = http.request(
        { host: 'localhost', port: BACKEND_PORT, path: req.url, method: req.method, headers: req.headers },
        (r) => {
          res.writeHead(r.statusCode, r.headers)
          r.pipe(res)
        },
      )
      upstream.on('error', () => {
        res.writeHead(502)
        res.end('backend down')
      })
      req.pipe(upstream)
    }, DELAY_MS)
  })
  .listen(LISTEN_PORT, () => console.log(`delay proxy on ${LISTEN_PORT}, ${DELAY_MS}ms`))
