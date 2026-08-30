import { defineConfig } from 'vitest/config';

// Node environment: the shared package is framework-agnostic and its runtime
// dependencies (fetch, Headers, Response) are Node globals on the supported
// version. Storage is injected per test via configureStorage().
export default defineConfig({
  test: {
    environment: 'node',
  },
});
