import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

function requireProductionEnv(mode: string) {
  if (mode !== 'production') {
    return
  }
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  const missing = ['VITE_STRIPE_PUBLISHABLE_KEY'].filter((key) => !env[key])
  if (missing.length > 0) {
    throw new Error(`Missing required production env vars: ${missing.join(', ')}`)
  }
}

export default defineConfig(({ mode }) => {
  requireProductionEnv(mode)
  return {
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  }
})
