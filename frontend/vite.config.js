import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss()
  ],
  build: {
    // Output directly to the Go server's static directory
    outDir: '../server/static',
    emptyOutDir: true
  },
  server: {
    // Proxy API requests to Go backend during local dev
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})