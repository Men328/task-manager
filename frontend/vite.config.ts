import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Dev proxy: the browser talks to the Vite dev server, which forwards
// /api/identity/*  -> http://localhost:8081/*  (identity service)
// /api/task/*      -> http://localhost:8082/*  (task service)
// /api/mail/*      -> http://localhost:8084/*  (mail-provider webhook)
// `rewrite` is the current Vite (§5/§6) API for rewriting the proxied path.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: false,
    proxy: {
      '/api/identity': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/identity/, ''),
      },
      '/api/task': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/task/, ''),
      },
      '/api/mail': {
        target: 'http://localhost:8084',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/mail/, ''),
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
