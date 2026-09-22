import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Dev proxy: the browser talks to the Vite dev server, which forwards
// /api/identity/*  -> http://localhost:8081/*  (identity service)
// /api/task/*      -> http://localhost:8082/*  (task service)
// /api/workspace/* -> http://localhost:8083/*  (workspace service)
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
      '/api/workspace': {
        target: 'http://localhost:8083',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/workspace/, ''),
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
