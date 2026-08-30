import { defineConfig } from 'vitest/config';
import path from 'node:path';

// Vitest runs in a jsdom environment so DOMPurify (which needs a DOM) works.
// The @cairn/shared alias mirrors vite.config.ts / tsconfig paths.
export default defineConfig({
  resolve: {
    alias: {
      '@cairn/shared': path.resolve(import.meta.dirname, '../shared/src/index.ts'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
  },
});
