import { defineConfig } from 'vitest/config';

// The shared package is framework-agnostic, so its tests run in plain Node with
// an in-memory StorageAdapter — no DOM and no React Native runtime needed.
export default defineConfig({
  test: {
    environment: 'node',
  },
});
