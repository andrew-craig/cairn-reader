import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import pkg from './package.json';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Expose the app's own version (from package.json) as a compile-time constant
  // so the About page can display it without bundling all of package.json.
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  resolve: {
    alias: {
      // Consume the shared workspace package directly from source (no build step).
      '@cairn/shared': path.resolve(import.meta.dirname, '../shared/src/index.ts'),
    },
  },
});
