# [MEDIUM] Fix type safety violations in Mobile App

Labels: enhancement, priority:medium, mobile, typescript

## Problem

The mobile app has **3 instances** of explicit `any` types that reduce TypeScript's type safety and ability to catch errors at compile time.

## Impact

- **Reduced Type Safety**: Using `any` bypasses TypeScript's type checking
- **Runtime Errors**: Type mismatches that could be caught at compile time may only surface at runtime
- **Poor IDE Support**: Autocomplete and refactoring tools work less effectively with `any` types

## Affected Files

### 1. ArticleListScreen.tsx:24
```typescript
// Current (line 24)
onArticlePress?: (article: any) => void;

// Should be
onArticlePress?: (article: Article) => void;
```

**File:** `src/components/ArticleListScreen.tsx:24`
**Fix:** Import `Article` type from `src/types/article` and use it instead of `any`

### 2. ArticleListScreen.tsx - onViewableItemsChanged (line 29-39)
```typescript
// Current (around line 35-38)
onViewableItemsChanged?: (info: any) => void;

// Should be
import { ViewToken } from 'react-native';

onViewableItemsChanged?: (info: {
  viewableItems: ViewToken[];
  changed: ViewToken[];
}) => void;
```

**File:** `src/components/ArticleListScreen.tsx`
**Fix:** Import `ViewToken` from `react-native` and use proper type definition

### 3. ExploreScreen.tsx:139
```typescript
// Current (line 139)
onViewableItemsChanged={(info: any) => {
  handleViewableItemsChanged(info);
}}

// Should be
import { ViewToken } from 'react-native';

onViewableItemsChanged={(info: {
  viewableItems: ViewToken[];
  changed: ViewToken[];
}) => {
  handleViewableItemsChanged(info);
}}
```

**File:** `src/screens/ExploreScreen.tsx:139`
**Fix:** Use proper ViewToken type from React Native

## Recommended Implementation

1. Add imports at the top of each file:
```typescript
import { ViewToken } from 'react-native';
import { Article } from '../types/article';
```

2. Update component props and function signatures to use proper types

3. Ensure ESLint warning for `@typescript-eslint/no-explicit-any` is enabled

## Testing

After fixes, verify with:
```bash
cd apps/mobile
npm run type-check  # Should pass without any type errors
npm run lint        # Should not show any warnings for explicit any
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "Type Safety Violations")
- ESLint Rule: `@typescript-eslint/no-explicit-any`
- TypeScript: [Avoid using any](https://www.typescriptlang.org/docs/handbook/declaration-files/do-s-and-don-ts.html#general-types)
- Estimated Effort: **1-2 hours**

## Acceptance Criteria

- [ ] All 3 instances of `any` replaced with proper types
- [ ] `Article` type is used for article parameters
- [ ] `ViewToken` type is used for viewable items changed handlers
- [ ] Type checking passes: `npm run type-check`
- [ ] ESLint shows no warnings for explicit any: `npm run lint`
- [ ] App builds and runs successfully
