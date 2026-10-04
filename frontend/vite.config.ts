import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Dev proxy: the browser talks to the Vite dev server, which forwards
// /api/identity/*  -> http://localhost:8081/*  (identity service)
// /api/task/*      -> http://localhost:8082/*  (task service)
// /api/calendar/*  -> http://localhost:8083/*  (calendar service)
// /api/event/*     -> http://localhost:8085/*  (event service)
// /api/backlog/*   -> http://localhost:8086/*  (backlog service)
// /api/report/*    -> http://localhost:8087/*  (report service)
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
      '/api/calendar': {
        target: 'http://localhost:8083',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/calendar/, ''),
      },
      '/api/event': {
        target: 'http://localhost:8085',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/event/, ''),
      },
      '/api/backlog': {
        target: 'http://localhost:8086',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/backlog/, ''),
      },
      '/api/report': {
        target: 'http://localhost:8087',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/report/, ''),
      },
      '/api/mail': {
        target: 'http://localhost:8084',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/mail/, ''),
      },
      '/api/notification': {
        target: 'http://localhost:8088',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/notification/, ''),
      },
      '/api/attachment': {
        target: 'http://localhost:8089',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/attachment/, ''),
      },
      // Soketi (Pusher protocol) giữ nguyên path /app/<key> sau khi bỏ prefix.
      '/api/soketi': {
        target: 'http://localhost:6001',
        changeOrigin: true,
        ws: true,
        rewrite: (path) => path.replace(/^\/api\/soketi/, ''),
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
