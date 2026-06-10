import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // Consume the shared workspace package directly from source (no build step).
      '@cairn/shared': path.resolve(import.meta.dirname, '../shared/src/index.ts'),
    },
  },
});
