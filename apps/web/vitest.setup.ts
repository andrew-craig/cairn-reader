import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// Unmount React trees rendered by @testing-library/react between tests.
afterEach(() => {
  cleanup();
});
