import { afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';

// Unmount React trees rendered by @testing-library/react between tests,
// and restore any mocks/spies so they don't leak into later tests.
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
