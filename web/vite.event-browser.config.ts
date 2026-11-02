import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vite'

// Builds only the event browser page. The Go CLI embeds dist-event-browser and
// serves it next to a local proxy, so this UI never ships with the main app.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist-event-browser',
    emptyOutDir: true,
    rollupOptions: {
      input: 'event-browser.html',
    },
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:7777',
    },
  },
})
