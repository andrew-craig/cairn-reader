# Add Mobile App Test Infrastructure

**Priority:** P3
**Status:** pending
**Task ID:** 18

## Problem

Mobile app has no test files (0 test coverage).

## Impact

No automated testing for mobile app components. Makes refactoring risky and bugs hard to catch. Reduces code quality and maintainability.

## Current Implementation

No test files exist in `apps/mobile/src/`.

## Proposed Solution

1. Install testing dependencies:
```bash
cd apps/mobile
npm install --save-dev @testing-library/react-native @testing-library/jest-native jest
```

2. Create `jest.config.js`:
```javascript
module.exports = {
  preset: 'react-native',
  setupFilesAfterEnv: ['<rootDir>/jest-setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!(react-native|@react-native|@react-navigation|expo)/)',
  ],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/types/**',
  ],
};
```

3. Create example test `src/components/common/__tests__/Button.test.tsx`:
```typescript
import React from 'react';
import { render, fireEvent } from '@testing-library/react-native';
import { Button } from '../Button';

describe('Button', () => {
  it('renders correctly', () => {
    const { getByText } = render(<Button title="Test" onPress={() => {}} />);
    expect(getByText('Test')).toBeTruthy();
  });

  it('calls onPress when pressed', () => {
    const mockOnPress = jest.fn();
    const { getByText } = render(<Button title="Test" onPress={mockOnPress} />);

    fireEvent.press(getByText('Test'));
    expect(mockOnPress).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    const mockOnPress = jest.fn();
    const { getByText } = render(
      <Button title="Test" onPress={mockOnPress} disabled />
    );

    fireEvent.press(getByText('Test'));
    expect(mockOnPress).not.toHaveBeenCalled();
  });
});
```

4. Add test script to `package.json`:
```json
{
  "scripts": {
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  }
}
```

5. Run tests:
```bash
npm test
```

## Files to Modify

- `apps/mobile/package.json` - add dependencies and scripts
- `apps/mobile/jest.config.js` - create
- `apps/mobile/jest-setup.js` - create
- `apps/mobile/src/components/**/__tests__/*.test.tsx` - add tests

## Testing

- Run `npm test` and verify tests execute
- Verify code coverage reports generate
- Verify watch mode works for development

## Implementation Notes

- Start with testing common components
- Add tests as new components are created
- Aim for 70%+ coverage for critical paths
- Use snapshot testing sparingly
