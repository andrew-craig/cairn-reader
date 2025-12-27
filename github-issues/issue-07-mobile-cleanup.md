# [LOW] Clean up unused code in Mobile App

Labels: cleanup, priority:low, mobile, typescript

## Problem

The mobile app has several instances of unused code including variables, imports, dependencies, and potentially unused exported types.

## Impact

- **Bundle Size**: Unused imports and dependencies increase app bundle size
- **Code Clarity**: Unused variables can confuse developers
- **Maintenance**: Dead code requires ongoing maintenance for no benefit
- **Install Time**: Unused npm dependencies increase `npm install` time

## Affected Areas

### 1. Unused Imports (4 instances)

**File:** `src/components/common/ArticleRow.tsx:11`

```typescript
// Current
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../../constants';

// Only Colors and FontFamily are used
// Spacing, FontSizes, and BorderRadius are imported but never used
```

**Fix:**
```typescript
import { Colors, FontFamily } from '../../constants';
```

### 2. Variable Should Be Const

**File:** `src/screens/ExploreScreen.tsx:72`

```typescript
// Current
let shouldContinue = true;  // Never reassigned

// Should be
const shouldContinue = true;  // Or remove if unnecessary
```

### 3. Unused Dependencies (1 package)

**File:** `package.json:27`

```json
"expo-linking": "~8.0.10"  // Installed but never imported
```

**Impact:** Adds ~100KB to node_modules

**Action Required:**
- Verify it's truly unused (search codebase for `expo-linking`)
- If unused, remove from package.json
- If planned for future use, add comment: `// Reserved for deep linking feature`

### 4. Unused File

**File:** `src/navigation/index.ts`

This file exports navigation components but is never imported anywhere.

**Fix:**
```bash
# Option 1: Remove if unnecessary
rm src/navigation/index.ts

# Option 2: Use it as the navigation entry point
# Import from this file instead of directly importing RootNavigator
```

### 5. Potentially Unused Exported Types (7 types)

These types may be intended for future backend integration:

**File:** `src/services/explore.ts`
- `RecommendationsResponse` (line 7)
- `BackendArticle` (line 13)
- `VoteRequest` (line 36)
- `MarkAsReadRequest` (line 36)

**File:** `src/types/article.ts`
- `ArticleMetadata` (line 18)
- `SortOption` (line 27)
- `FilterOption` (line 28)

**Action Required:**
- If these are for future backend integration, add JSDoc comments:
  ```typescript
  /**
   * Backend article response format.
   * @todo Used when backend integration is complete
   */
  export interface BackendArticle {
    // ...
  }
  ```
- If truly unused, remove them

## Recommended Actions (Priority Order)

### Priority 1: Unused Imports (Quick win)
```bash
cd apps/mobile
# Fix automatically with ESLint
npx eslint --fix src/components/common/ArticleRow.tsx
```

### Priority 2: Variable Declaration Style
Configure ESLint to catch this automatically:
```javascript
// .eslintrc.js
rules: {
  'prefer-const': 'warn',
}
```

### Priority 3: Unused Dependencies
```bash
cd apps/mobile
npm uninstall expo-linking  # Only if truly unused
npm install  # Update lockfile
```

### Priority 4: Document or Remove Types
Add comments to types reserved for future use, or remove unused ones.

### Priority 5: Remove Unused File
```bash
rm src/navigation/index.ts  # If confirmed unused
```

## Testing

After cleanup, verify:
```bash
cd apps/mobile

# Type checking still passes
npm run type-check

# Linting passes
npm run lint

# App builds successfully
npm start
# Press 'i' for iOS or 'a' for Android

# App functions correctly
# Test: Navigation, article loading, etc.
```

## Automation

Configure ESLint to auto-fix on save:

**File:** `.vscode/settings.json` (create if doesn't exist)
```json
{
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "eslint.validate": [
    "typescript",
    "typescriptreact"
  ]
}
```

Add to ESLint config:

**File:** `apps/mobile/.eslintrc.js`
```javascript
module.exports = {
  extends: ['expo', 'plugin:@typescript-eslint/recommended'],
  parser: '@typescript-eslint/parser',
  plugins: ['@typescript-eslint'],
  rules: {
    '@typescript-eslint/no-unused-vars': ['warn', {
      argsIgnorePattern: '^_',
      varsIgnorePattern: '^_'
    }],
    '@typescript-eslint/no-explicit-any': 'warn',
    'prefer-const': 'warn',  // Add this
  },
};
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Multiple sections)
- Tools: Knip, ESLint, depcheck
- Estimated Effort: **1 hour**

## Acceptance Criteria

- [ ] All unused imports removed from ArticleRow.tsx
- [ ] `shouldContinue` variable changed to const or refactored
- [ ] Decision made on `expo-linking` dependency (remove or document)
- [ ] Decision made on `src/navigation/index.ts` (remove or use)
- [ ] Unused exported types either documented or removed
- [ ] ESLint configured to catch these issues automatically
- [ ] All tests pass: `npm run type-check && npm run lint`
- [ ] App builds and runs successfully

## Notes

- This is low priority cleanup that can be done incrementally
- Consider enabling ESLint auto-fix to prevent future issues
- Some "unused" types may be intentionally defined for future backend integration
