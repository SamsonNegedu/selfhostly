/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly DEV?: boolean
  readonly PROD?: boolean
  readonly MODE?: string
  readonly SSR?: boolean
  readonly VITE_?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
