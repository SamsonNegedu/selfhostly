import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

// Words the gateway routes on. Keep in step with internal/gateway/router.go and scripts/check-asset-names.mjs.
const GATEWAY_WORDS = /api|auth|avatar/gi

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
        '/auth': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          // The tunnel's ingress rules send any path that matches /api, /auth or /avatar to the gateway, and those
          // rules match anywhere in the path: "/assets/api-<hash>.js" and even "chunk-api-<hash>.js" are caught by
          // a loose one. So no file name may contain those words at all. scripts/check-asset-names.mjs fails the
          // build if one slips through.
          entryFileNames: 'assets/app-[name]-[hash].js',
          chunkFileNames: (chunk) => `assets/chunk-${chunk.name.replace(GATEWAY_WORDS, 'x')}-[hash].js`,
        },
      },
    },
    define: {
      // Make environment variables available in the browser
      'import.meta.env.VITE_API_BASE': JSON.stringify(env.VITE_API_BASE || ''),
    },
  }
})
